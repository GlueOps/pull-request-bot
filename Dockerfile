FROM python:3.14.0-alpine@sha256:e1a567200b6d518567cc48994d3ab4f51ca54ff7f6ab0f3dc74ac5c762db0800

RUN pip install --upgrade pip

COPY requirements.txt /app/requirements.txt
WORKDIR /app
RUN pip install -r requirements.txt
COPY main.py /app/main.py
COPY ./src /app/src

CMD [ "python", "-u", "main.py" ]
