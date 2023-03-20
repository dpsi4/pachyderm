import io
from typing import Callable

from tests.fixtures import *
from tests.utils import count

from pachyderm_sdk.api import pfs, pps

IMAGE_NAME = "bonenfan5ben/datum_batching:0035"


def generate_std_in(func: Callable[[], None]):
    """Generates the std_in field of for the test pipelines.

    Args:
        func: The function containing the "user code" of the test pipeline.
          This must be defined at the root of the script, i.e. not as a
          method to a class or defined within another function.
    """
    from inspect import getsource

    test_script = (
        f'{getsource(func)}\n\n'
        'if __name__ == "__main__":\n'
        f'    {func.__name__}()\n'
    )
    return [
        f"echo '{test_script}' > main.py",
        "python3 main.py",
    ]


def user_code():
    """Assert the one file is mounted in /pfs/batch_datums_input
    and copy it to /pfs/out.
    """
    import os
    import shutil
    from pachyderm_sdk import Client

    while True:
        with Client.worker.batch_datums():
            datum_files = os.listdir(f"/pfs/batch_datums_input")
            print(datum_files)
            assert len(datum_files) == 1
            shutil.copy(f"/pfs/batch_datums_input/{datum_files[0]}", f"/pfs/out/{datum_files[0]}")


def test_datum_batching(client: TestClient):
    """Test that exceptions within the user code is caught, reported to the
    worker binary, and iteration continues.

    This test uploads 10 files to the input repo and the user code
      copies each file to the output repo. The pipeline job should finish
      successfully and the output repo should contain 10 files.
    """
    repo = client.new_repo()
    branch = pfs.Branch(repo=repo, name="master")

    input_files = [f"/file_{i:02d}.dat" for i in range(10)]
    with client.pfs.commit(branch=branch) as commit:
        for file in input_files:
            commit.put_file_from_file(path=file, file=io.BytesIO(b"DATA"))

    pipeline = pps.Pipeline(name=client._generate_name())
    try:
        client.pps.create_pipeline(
            pipeline=pipeline,
            input=pps.Input(
                pfs=pps.PfsInput(
                    project=repo.project.name,
                    repo=repo.name,
                    name="batch_datums_input",  # Name explicitly set for user code.
                    glob="/*",
                )
            ),
            transform=pps.Transform(
                cmd=["bash", ],
                datum_batching=True,
                image=IMAGE_NAME,
                stdin=generate_std_in(user_code)
            )
        )
        job_info = next(client.pps.list_job(pipeline=pipeline))
        client.pps.inspect_job(job=job_info.job, wait=True)

        output_files = client.pfs.list_file(file=pfs.File(commit=job_info.output_commit))
        assert count(output_files) == count(input_files)
    finally:  # Cleanup our manually defined test pipeline.
        if client.pps.pipeline_exists(pipeline):
            client.pps.delete_pipeline(pipeline=pipeline, force=True)


def user_code_errors():
    """Raises an Exception for every datum."""
    from pachyderm_sdk import Client

    while True:
        with Client.worker.batch_datums():
            raise Exception("Something Bad Happened!")


def test_datum_batching_errors(client: TestClient):
    """Test that exceptions within the user code is caught, reported to the
    worker binary, and iteration continues.

    This test uploads a single file to the input repo and the user code
      always raises an exception. The pipeline has err_cmd set to "true", so
      the pipeline job should finish successfully and the output repo should
      be empty.
    """
    repo = client.new_repo()
    branch = pfs.Branch(repo=repo, name="master")
    with client.pfs.commit(branch=branch) as commit:
        commit.put_file_from_file(path="/file.dat", file=io.BytesIO(b"DATA"))

    pipeline = pps.Pipeline(name=client._generate_name())
    try:
        client.pps.create_pipeline(
            pipeline=pipeline,
            input=pps.Input(
                pfs=pps.PfsInput(
                    project=repo.project.name,
                    repo=repo.name,
                    glob="/*",
                )
            ),
            transform=pps.Transform(
                cmd=["bash", ],
                err_cmd=["true", ],  # Note err_cmd set.
                datum_batching=True,
                image=IMAGE_NAME,
                stdin=generate_std_in(user_code_errors)
            )
        )
        started_job = next(client.pps.list_job(pipeline=pipeline))
        completed_job = client.pps.inspect_job(job=started_job.job, wait=True)
        assert completed_job.state == pps.JobState.JOB_SUCCESS

        output_files = client.pfs.list_file(file=pfs.File(commit=completed_job.output_commit))
        assert count(output_files) == 0
    finally:  # Cleanup our manually defined test pipeline.
        if client.pps.pipeline_exists(pipeline):
            client.pps.delete_pipeline(pipeline=pipeline, force=True)
