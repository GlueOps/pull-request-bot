FROM python:3.13.4-alpine@sha256:b4d299311845147e7e47c970566906caf8378a1f04e5d3de65b5f2e834f8e3bf

RUN pip install --upgrade pip

COPY requirements.txt /app/requirements.txt
WORKDIR /app
RUN pip install -r requirements.txt
COPY main.py /app/main.py
COPY ./src /app/src

CMD [ "python", "-u", "main.py" ]
