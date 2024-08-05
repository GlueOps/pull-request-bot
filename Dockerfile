FROM python:3.11.9-alpine@sha256:700b4aa84090748aafb348fc042b5970abb0a73c8f1b4fcfe0f4e3c2a4a9fcca

RUN pip install --upgrade pip

COPY requirements.txt /app/requirements.txt
WORKDIR /app
RUN pip install -r requirements.txt
COPY main.py /app/main.py
COPY ./src /app/src

CMD [ "python", "-u", "main.py" ]
