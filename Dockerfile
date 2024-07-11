FROM python:3.11.9-alpine@sha256:eb8afe385f10ccc9da8378e8712fea8b26f4ed009648f8afea4518836d8b3ed3

RUN pip install --upgrade pip

COPY requirements.txt /app/requirements.txt
WORKDIR /app
RUN pip install -r requirements.txt
COPY main.py /app/main.py
COPY ./src /app/src

CMD [ "python", "-u", "main.py" ]
