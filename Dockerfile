FROM python:3.13.1-alpine@sha256:5dad625efcbc6fad19c10b7b2bfefa1c7a8129c8f8343106b639c27dd9e7db2c

RUN pip install --upgrade pip

COPY requirements.txt /app/requirements.txt
WORKDIR /app
RUN pip install -r requirements.txt
COPY main.py /app/main.py
COPY ./src /app/src

CMD [ "python", "-u", "main.py" ]
