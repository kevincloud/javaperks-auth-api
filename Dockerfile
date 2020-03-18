FROM ubuntu:18.04
  
LABEL Kevin Cochran "kcochran@hashicorp.com"

RUN apt-get update && apt-get upgrade -y
RUN apt-get install -y npm
RUN mkdir /app
ADD /tmp/javaperks-auth-api /app/

WORKDIR /app

CMD [ "javaperks-auth-api" ]
