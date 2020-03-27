INSERT INTO public.client (peer_id, serial, psk, healthy, last_check) VALUES (1, 'serial', 'pskkk', true, '2020-03-26 12:26:32.000000');
INSERT INTO public.gateway (peer_id, id, access_group_id, endpoint) VALUES (2, 1, '1234-asdf-aad', '1.2.3.4:1234');
INSERT INTO public.ip (peer_id, ip) VALUES (1, '10.1.1.1');
INSERT INTO public.ip (peer_id, ip) VALUES (2, '10.1.1.2');
INSERT INTO public.peer (id, public_key, kind) VALUES (1, 'publickey', 'client');
INSERT INTO public.peer (id, public_key, kind) VALUES (2, 'pk-gw', 'gateway');
INSERT INTO public.routes (gateway_id, cidr) VALUES (1, '1.2.3.4/23');