FROM davidwalter/debian-stretch-slim
COPY bin/llb /opt/sbin/llb
CMD [ "/opt/sbin/llb" ]
