# Catalogue DB: MongoDB base image with security hardening
# Note: In production, use an official MongoDB image
FROM mongo:4.4

# Security: MongoDB data directory permissions
RUN chown -R mongodb:mongodb /data/db

EXPOSE 27017

CMD ["mongod", "--bind_ip_all", "--auth"]
