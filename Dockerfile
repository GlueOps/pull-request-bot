FROM python:3.14.2-alpine@sha256:f74e244c26cf94c81a2a6ec8e4e5e55e59bae979063c83382cafb87f03fc1f56

RUN pip install --upgrade pip

COPY requirements.txt /app/requirements.txt
WORKDIR /app
RUN pip install -r requirements.txt
COPY main.py /app/main.py
COPY ./src /app/src

CMD [ "python", "-u", "main.py" ]
