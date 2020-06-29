FROM php:7.4-cli-alpine AS build
WORKDIR /app
COPY composer.json composer.lock src templates ./
RUN php -r "copy('https://getcomposer.org/installer', 'composer-setup.php');" && \
    EXPECTED_SIGNATURE="$(wget -q -O - https://composer.github.io/installer.sig)" && \
    ACTUAL_SIGNATURE="$(php -r "echo hash_file('sha384', 'composer-setup.php');")" && \
    [[ "$ACTUAL_SIGNATURE" == "$EXPECTED_SIGNATURE" ]] || { echo >&2 "Corrupt installer"; exit 1; } && \
    php composer-setup.php && \
    php -r "unlink('composer-setup.php');"
RUN php composer.phar install -o --no-dev --no-plugins --no-scripts --no-ansi --no-progress

FROM php:7.4-apache
RUN a2enmod rewrite
RUN echo "export ISSUER_ENTITY_ID" >> /etc/apache2/envvars && \
    echo "export LOGIN_URL" >> /etc/apache2/envvars && \
    echo "export ACCESS_GROUP" >> /etc/apache2/envvars && \
    echo "export AAD_CLIENT_ID" >> /etc/apache2/envvars && \
    echo "export SAML_CERT" >> /etc/apache2/envvars && \
    echo "export AAD_CLIENT_SECRET" >> /etc/apache2/envvars && \
    echo "export DOMAIN" >> /etc/apache2/envvars
COPY --from=build /app/vendor/ /var/www/vendor/
COPY templates/ /var/www/templates/
COPY src/ /var/www/src/
COPY public/ /var/www/html/
