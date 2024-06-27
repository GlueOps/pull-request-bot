FROM python:3.12.4-alpine@sha256:ff870bf7c2bb546419aaea570f0a1c28c8103b78743a2b8030e9e97391ddf81b

RUN pip install --upgrade pip

COPY requirements.txt /app/requirements.txt
WORKDIR /app
RUN pip install -r requirements.txt
COPY main.py /app/main.py
COPY ./src /app/src

CMD [ "python", "-u", "main.py" ]
