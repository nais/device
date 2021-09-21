--
-- PostgreSQL database dump
--

-- Dumped from database version 12.8
-- Dumped by pg_dump version 13.4

SET statement_timeout = 0;
SET lock_timeout = 0;
SET idle_in_transaction_session_timeout = 0;
SET client_encoding = 'UTF8';
SET standard_conforming_strings = on;
SELECT pg_catalog.set_config('search_path', '', false);
SET check_function_bodies = false;
SET xmloption = content;
SET client_min_messages = warning;
SET row_security = off;

--
-- Name: platform; Type: TYPE; Schema: public; Owner: postgres
--

CREATE TYPE public.platform AS ENUM (
    'darwin',
    'linux',
    'windows'
);


ALTER TYPE public.platform OWNER TO postgres;

SET default_tablespace = '';

SET default_table_access_method = heap;

--
-- Name: device; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.device (
    id integer NOT NULL,
    username character varying,
    serial character varying,
    psk character varying(44),
    platform public.platform,
    healthy boolean,
    public_key character varying(44) NOT NULL,
    ip character varying(15),
    last_updated timestamp with time zone,
    kolide_last_seen timestamp with time zone
);


ALTER TABLE public.device OWNER TO postgres;

--
-- Name: device_id_seq; Type: SEQUENCE; Schema: public; Owner: postgres
--

CREATE SEQUENCE public.device_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


ALTER TABLE public.device_id_seq OWNER TO postgres;

--
-- Name: device_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: postgres
--

ALTER SEQUENCE public.device_id_seq OWNED BY public.device.id;


--
-- Name: gateway; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.gateway (
    id integer NOT NULL,
    name character varying NOT NULL,
    access_group_ids character varying DEFAULT ''::character varying,
    endpoint character varying(21),
    public_key character varying(44) NOT NULL,
    ip character varying(15),
    routes character varying DEFAULT ''::character varying,
    requires_privileged_access boolean DEFAULT false
);


ALTER TABLE public.gateway OWNER TO postgres;

--
-- Name: gateway_id_seq; Type: SEQUENCE; Schema: public; Owner: postgres
--

CREATE SEQUENCE public.gateway_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


ALTER TABLE public.gateway_id_seq OWNER TO postgres;

--
-- Name: gateway_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: postgres
--

ALTER SEQUENCE public.gateway_id_seq OWNED BY public.gateway.id;


--
-- Name: migrations; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.migrations (
    version integer NOT NULL,
    created timestamp with time zone NOT NULL
);


ALTER TABLE public.migrations OWNER TO postgres;

--
-- Name: session; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.session (
    key character varying,
    device_id integer,
    groups character varying,
    object_id character varying,
    expiry timestamp with time zone
);


ALTER TABLE public.session OWNER TO postgres;

--
-- Name: device id; Type: DEFAULT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.device ALTER COLUMN id SET DEFAULT nextval('public.device_id_seq'::regclass);


--
-- Name: gateway id; Type: DEFAULT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.gateway ALTER COLUMN id SET DEFAULT nextval('public.gateway_id_seq'::regclass);


--
-- Name: device device_ip_key; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.device
    ADD CONSTRAINT device_ip_key UNIQUE (ip);


--
-- Name: device device_pkey; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.device
    ADD CONSTRAINT device_pkey PRIMARY KEY (id);


--
-- Name: device device_public_key_key; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.device
    ADD CONSTRAINT device_public_key_key UNIQUE (public_key);


--
-- Name: device device_serial_platform_key; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.device
    ADD CONSTRAINT device_serial_platform_key UNIQUE (serial, platform);


--
-- Name: gateway gateway_ip_key; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.gateway
    ADD CONSTRAINT gateway_ip_key UNIQUE (ip);


--
-- Name: gateway gateway_name_key; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.gateway
    ADD CONSTRAINT gateway_name_key UNIQUE (name);


--
-- Name: gateway gateway_pkey; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.gateway
    ADD CONSTRAINT gateway_pkey PRIMARY KEY (id);


--
-- Name: gateway gateway_public_key_key; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.gateway
    ADD CONSTRAINT gateway_public_key_key UNIQUE (public_key);


--
-- Name: migrations migrations_pkey; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.migrations
    ADD CONSTRAINT migrations_pkey PRIMARY KEY (version);


--
-- Name: device_lower_case_username; Type: INDEX; Schema: public; Owner: postgres
--

CREATE INDEX device_lower_case_username ON public.device USING btree (lower((username)::text));


--
-- Name: expiry_idx; Type: INDEX; Schema: public; Owner: postgres
--

CREATE INDEX expiry_idx ON public.session USING btree (expiry);


--
-- Name: session_key_idx; Type: INDEX; Schema: public; Owner: postgres
--

CREATE UNIQUE INDEX session_key_idx ON public.session USING btree (key);


--
-- Name: session session_device_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.session
    ADD CONSTRAINT session_device_id_fkey FOREIGN KEY (device_id) REFERENCES public.device(id);


--
-- PostgreSQL database dump complete
--

