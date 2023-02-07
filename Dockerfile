FROM python:3.11.1

COPY requirements.txt /app/requirements.txt
WORKDIR /app
RUN pip install -r requirements.txt
COPY main.py /app/main.py

CMD [ "python", "main.py" ]