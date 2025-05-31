FROM python:3.11.12-alpine@sha256:a648a482d0124da939ead54c5c6f0f6ce0b4ac925749c7d9ad3c2eba838966f1

RUN pip install --upgrade pip

COPY requirements.txt /app/requirements.txt
WORKDIR /app
RUN pip install -r requirements.txt
COPY main.py /app/main.py
COPY ./src /app/src

CMD [ "python", "-u", "main.py" ]
