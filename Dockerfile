FROM python:3.14.1-alpine@sha256:b80c82b1a282283bd3e3cd3c6a4c895d56d1385879c8c82fa673e9eb4d6d4aa5

RUN pip install --upgrade pip

COPY requirements.txt /app/requirements.txt
WORKDIR /app
RUN pip install -r requirements.txt
COPY main.py /app/main.py
COPY ./src /app/src

CMD [ "python", "-u", "main.py" ]
