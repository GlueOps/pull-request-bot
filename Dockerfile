FROM python:3.11.11-alpine@sha256:9ae1ab261b73eeaf88957c42744b8ec237faa8fa0d5be22a3ed697f52673b58a

RUN pip install --upgrade pip

COPY requirements.txt /app/requirements.txt
WORKDIR /app
RUN pip install -r requirements.txt
COPY main.py /app/main.py
COPY ./src /app/src

CMD [ "python", "-u", "main.py" ]
