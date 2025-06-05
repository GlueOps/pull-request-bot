FROM python:3.11.13-alpine@sha256:8068890a42d68ece5b62455ef327253249b5f094dcdee57f492635a40217f6a3

RUN pip install --upgrade pip

COPY requirements.txt /app/requirements.txt
WORKDIR /app
RUN pip install -r requirements.txt
COPY main.py /app/main.py
COPY ./src /app/src

CMD [ "python", "-u", "main.py" ]
