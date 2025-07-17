FROM python:3.12.11-alpine@sha256:9b8808206f4a956130546a32cbdd8633bc973b19db2923b7298e6f90cc26db08

RUN pip install --upgrade pip

COPY requirements.txt /app/requirements.txt
WORKDIR /app
RUN pip install -r requirements.txt
COPY main.py /app/main.py
COPY ./src /app/src

CMD [ "python", "-u", "main.py" ]
