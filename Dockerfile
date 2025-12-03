FROM python:3.14.1-alpine@sha256:4f699e4afac838c50be76deac94a6dde0e287d5671fd8e95eb410f850801b237

RUN pip install --upgrade pip

COPY requirements.txt /app/requirements.txt
WORKDIR /app
RUN pip install -r requirements.txt
COPY main.py /app/main.py
COPY ./src /app/src

CMD [ "python", "-u", "main.py" ]
