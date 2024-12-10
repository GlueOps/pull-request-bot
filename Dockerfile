FROM python:3.12.8-alpine@sha256:b0fc5cb1a4ae39af99c0ddf4b56cb06e8f867dce47fa9a8553f8601e527596b4

RUN pip install --upgrade pip

COPY requirements.txt /app/requirements.txt
WORKDIR /app
RUN pip install -r requirements.txt
COPY main.py /app/main.py
COPY ./src /app/src

CMD [ "python", "-u", "main.py" ]
