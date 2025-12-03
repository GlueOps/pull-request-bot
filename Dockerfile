FROM python:3.14.1-alpine@sha256:eb37f58646a901dc7727cf448cae36daaefaba79de33b5058dab79aa4c04aefb

RUN pip install --upgrade pip

COPY requirements.txt /app/requirements.txt
WORKDIR /app
RUN pip install -r requirements.txt
COPY main.py /app/main.py
COPY ./src /app/src

CMD [ "python", "-u", "main.py" ]
