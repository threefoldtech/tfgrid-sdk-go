FROM alpine:3.17.2

RUN apk add --no-cache git make musl-dev go
RUN apk add --update nodejs npm

RUN apk add caddy

RUN apk add python3 py3-pip



RUN wget https://github.com/rawdaGastan/ginit/releases/download/v0.1/ginit_0.1_Linux_x86_64.tar.gz
RUN tar xzf ginit_0.1_Linux_x86_64.tar.gz && mv ginit /bin && rm ginit_0.1_Linux_x86_64.tar.gz

COPY ./init.sh .
RUN [ "chmod", "+x", "/init.sh"]
ENTRYPOINT [ "/init.sh" ]
