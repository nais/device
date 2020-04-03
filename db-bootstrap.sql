/* apiserver */
INSERT INTO public.peer (id, public_key, ip, type)
VALUES (1, 'FUwVtyvs8nIRx9RpUUEopkfV8idmHz9g9K/vf9MFOXI=', '10.255.240.1', 'control');

/* vegar */
INSERT INTO public.client (id, serial, psk, healthy, last_check)
VALUES (1, 'serial1', 'psk1', true, '2020-03-26 12:26:32.000000');
INSERT INTO public.peer (id, public_key, ip, type)
VALUES (3, 'EatjldYVvB91aep5kxDnYsQ37Ufk92IBBIcfma1fzAs=', '10.255.240.2', 'control');
INSERT INTO public.client_peer(client_id, peer_id)
VALUES (1, 3);
/* INSERT INTO public.peer (id, public_key, ip, type) VALUES (4, '', '10.255.248.2', 'data'); */
/* INSERT INTO public.client_peer(client_id, peer_id) VALUES (1, 4); */

/* johnny */
INSERT INTO public.client (id, serial, psk, healthy, last_check)
VALUES (2, 'serial2', 'psk2', true, '2020-03-26 12:26:32.000000');
INSERT INTO public.peer (id, public_key, ip, type)
VALUES (5, '', '10.255.240.3', 'control');
INSERT INTO public.client_peer(client_id, peer_id)
VALUES (2, 5);
/* INSERT INTO public.peer (id, public_key, ip, type) VALUES (6, '', '10.255.248.3', 'data'); */
/* INSERT INTO public.client_peer(client_id, peer_id) VALUES (2, 6); */


/* gateway 1 */
INSERT INTO public.gateway (id, access_group_id, endpoint)
VALUES (1, '1234-asdf-aad1', '35.228.118.232:51820');
INSERT INTO public.peer (id, public_key, ip, type)
VALUES (7, 'QFwvy4pUYXpYm4z9iXw1GZRgjp3iU+3Hsu0UUvre9FM=', '10.255.240.4', 'control');
INSERT INTO public.peer (id, public_key, ip, type)
VALUES (8, '55h6JA2ZMPzaoa+iZU62JmqmtgK3ydj4YdT9HkkhnEQ=', '10.255.248.4', 'data');
INSERT INTO public.gateway_peer(gateway_id, peer_id)
VALUES (1, 7);
INSERT INTO public.gateway_peer(gateway_id, peer_id)
VALUES (1, 8);

/* gateway 2 */
INSERT INTO public.gateway (id, access_group_id, endpoint)
VALUES (2, '1234-asdf-aad2', '35.228.118.232:51820');
INSERT INTO public.peer (id, public_key, ip, type)
VALUES (9, 'Whbuh2+T8/m1kJTtByfYQvlD/Efv4xxX9rbe9B2SK2M=', '10.255.240.5', 'control');
INSERT INTO public.peer (id, public_key, ip, type)
VALUES (10, 'i5AmQLLlPa4fQmfuHj7COCFwmwegI39WMfs/LIdzbFo=', '10.255.248.5', 'data');
INSERT INTO public.gateway_peer(gateway_id, peer_id)
VALUES (2, 9);
INSERT INTO public.gateway_peer(gateway_id, peer_id)
VALUES (2, 10);

INSERT INTO public.routes (gateway_id, cidr)
VALUES (1, '1.2.3.4/23');

