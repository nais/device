BEGIN;

INSERT INTO device (serial, username, psk, platform, healthy, public_key, ip)
VALUES ('C02VX5VUHTD6', 'vegar.sechmann.molvig@nav.no', 'psk1', 'darwin', true,
        'EatjldYVvB91aep5kxDnYsQ37Ufk92IBBIcfma1fzAs=',
        '10.255.240.2');

INSERT INTO device (serial, username, psk, platform, healthy, public_key, ip)
VALUES ('serial2', 'johnny.horvi@nav.no', 'psk2', 'linux', true, 'EatjldYVvB91aep5kxDnYsQ37Ufk92IBBIcfma1fzBB=',
        '10.255.240.3');
INSERT INTO device (serial, username, psk, platform, healthy, public_key, ip)
VALUES ('serial2', 'johnny.horvi@nav.no', 'psk2', 'darwin', true, 'EatjldYVvB91aep5kxDnYsQ37Ufk92IBBIcfma1fzAA=',
        '10.255.240.4');

INSERT INTO gateway (name, public_key, ip, endpoint, routes, access_group_ids)
VALUES ('gateway-1', 'QFwvy4pUYXpYm4z9iXw1GZRgjp3iU+3Hsu0UUvre9FM=', '10.255.240.4', '35.228.118.232:51820',
        '13.37.13.37/32', 'asd-123');

INSERT INTO gateway (name, public_key, ip, endpoint, routes, access_group_ids)
VALUES ('gateway-2', 'Whbuh2+T8/m1kJTtByfYQvlD/Efv4xxX9rbe9B2SK2M=', '10.255.240.5', '35.228.118.232:51820',
        '13.37.13.38/32,13.37.13.39/32', '123-asd');

END;
