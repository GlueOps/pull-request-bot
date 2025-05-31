FROM python:3.13.3-alpine@sha256:a94caf6aab428e086bc398beaf64a6b7a0fad4589573462f52362fd760e64cc9

RUN pip install --upgrade pip

COPY requirements.txt /app/requirements.txt
WORKDIR /app
RUN pip install -r requirements.txt
COPY main.py /app/main.py
COPY ./src /app/src

CMD [ "python", "-u", "main.py" ]
