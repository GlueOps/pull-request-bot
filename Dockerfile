FROM python:3.11.9-alpine@sha256:e8effbb94ea2e5439c6b69c97c6455ff11fce94b7feaaed84bb0f2300d798cb7

RUN pip install --upgrade pip

COPY requirements.txt /app/requirements.txt
WORKDIR /app
RUN pip install -r requirements.txt
COPY main.py /app/main.py
COPY ./src /app/src

CMD [ "python", "-u", "main.py" ]
