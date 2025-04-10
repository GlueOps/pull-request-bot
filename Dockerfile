FROM python:3.11.12-alpine@sha256:32ac7ba3dad4bcee9c8cfaf3b489f832b84ba0a1eb8ef76685456d424baaf444

RUN pip install --upgrade pip

COPY requirements.txt /app/requirements.txt
WORKDIR /app
RUN pip install -r requirements.txt
COPY main.py /app/main.py
COPY ./src /app/src

CMD [ "python", "-u", "main.py" ]
