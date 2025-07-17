FROM python:3.11.13-alpine@sha256:a25e12e5f7bd9ce4578bc87eadec231cc3aa7d6a03723601d3e6f82639969d3a

RUN pip install --upgrade pip

COPY requirements.txt /app/requirements.txt
WORKDIR /app
RUN pip install -r requirements.txt
COPY main.py /app/main.py
COPY ./src /app/src

CMD [ "python", "-u", "main.py" ]
