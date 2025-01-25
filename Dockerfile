FROM python:3.12.8-alpine@sha256:ba13ef990f6e5d13014e9e8d04c02a8fdb0fe53d6dccf6e19147f316e6cc3a84

RUN pip install --upgrade pip

COPY requirements.txt /app/requirements.txt
WORKDIR /app
RUN pip install -r requirements.txt
COPY main.py /app/main.py
COPY ./src /app/src

CMD [ "python", "-u", "main.py" ]
