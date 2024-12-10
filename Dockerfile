FROM python:3.11.11-alpine@sha256:bc84eb94541f34a0e98535b130ea556ae85f6a431fdb3095762772eeb260ffc3

RUN pip install --upgrade pip

COPY requirements.txt /app/requirements.txt
WORKDIR /app
RUN pip install -r requirements.txt
COPY main.py /app/main.py
COPY ./src /app/src

CMD [ "python", "-u", "main.py" ]
