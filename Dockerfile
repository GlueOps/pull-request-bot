FROM python:3.11.9-alpine@sha256:1bcefb95bd059ea0240d2fe86a994cf13ab7465d00836871cf1649bb2e98fb9f

RUN pip install --upgrade pip

COPY requirements.txt /app/requirements.txt
WORKDIR /app
RUN pip install -r requirements.txt
COPY main.py /app/main.py
COPY ./src /app/src

CMD [ "python", "-u", "main.py" ]
