FROM python:3.11.12-alpine@sha256:d0199977fdae5d1109a89d0b0014468465e014a9834d0a566ea50871b3255ade

RUN pip install --upgrade pip

COPY requirements.txt /app/requirements.txt
WORKDIR /app
RUN pip install -r requirements.txt
COPY main.py /app/main.py
COPY ./src /app/src

CMD [ "python", "-u", "main.py" ]
