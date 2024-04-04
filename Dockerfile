FROM python:3.11.9-alpine@sha256:506861259a53e68b95992ff711dd2aab9ff8dc8a50ff4dca24c6e88dc461563e

RUN pip install --upgrade pip

COPY requirements.txt /app/requirements.txt
WORKDIR /app
RUN pip install -r requirements.txt
COPY main.py /app/main.py
COPY ./src /app/src

CMD [ "python", "-u", "main.py" ]
