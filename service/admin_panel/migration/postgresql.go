package migration

import (
	"database/sql"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	"github.com/sagernet/sing-box/common/migrate/source"
)

var migrations = map[string]string{
	"1_initialize_schema.up.sql": `
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

        CREATE SEQUENCE public.goadmin_menu_myid_seq
            START WITH 1
            INCREMENT BY 1
            NO MINVALUE
            MAXVALUE 99999999
            CACHE 1;


        SET default_tablespace = '';

        SET default_table_access_method = heap;

        CREATE TABLE public.goadmin_menu (
            id integer DEFAULT nextval('public.goadmin_menu_myid_seq'::regclass) NOT NULL,
            parent_id integer DEFAULT 0 NOT NULL,
            type integer DEFAULT 0,
            "order" integer DEFAULT 0 NOT NULL,
            title character varying(50) NOT NULL,
            header character varying(100),
            plugin_name character varying(100) NOT NULL,
            icon character varying(50) NOT NULL,
            uri character varying(3000) NOT NULL,
            uuid character varying(100),
            created_at timestamp without time zone DEFAULT now(),
            updated_at timestamp without time zone DEFAULT now()
        );

        CREATE SEQUENCE public.goadmin_operation_log_myid_seq
            START WITH 1
            INCREMENT BY 1
            NO MINVALUE
            MAXVALUE 99999999
            CACHE 1;

        CREATE TABLE public.goadmin_operation_log (
            id integer DEFAULT nextval('public.goadmin_operation_log_myid_seq'::regclass) NOT NULL,
            user_id integer NOT NULL,
            path character varying(255) NOT NULL,
            method character varying(10) NOT NULL,
            ip character varying(15) NOT NULL,
            input text NOT NULL,
            created_at timestamp without time zone DEFAULT now(),
            updated_at timestamp without time zone DEFAULT now()
        );

        CREATE SEQUENCE public.goadmin_permissions_myid_seq
            START WITH 1
            INCREMENT BY 1
            NO MINVALUE
            MAXVALUE 99999999
            CACHE 1;

        CREATE TABLE public.goadmin_permissions (
            id integer DEFAULT nextval('public.goadmin_permissions_myid_seq'::regclass) NOT NULL,
            name character varying(50) NOT NULL,
            slug character varying(50) NOT NULL,
            http_method character varying(255),
            http_path text NOT NULL,
            created_at timestamp without time zone DEFAULT now(),
            updated_at timestamp without time zone DEFAULT now()
        );

        CREATE TABLE public.goadmin_role_menu (
            role_id integer NOT NULL,
            menu_id integer NOT NULL,
            created_at timestamp without time zone DEFAULT now(),
            updated_at timestamp without time zone DEFAULT now()
        );

        CREATE TABLE public.goadmin_role_permissions (
            role_id integer NOT NULL,
            permission_id integer NOT NULL,
            created_at timestamp without time zone DEFAULT now(),
            updated_at timestamp without time zone DEFAULT now()
        );

        CREATE TABLE public.goadmin_role_users (
            role_id integer NOT NULL,
            user_id integer NOT NULL,
            created_at timestamp without time zone DEFAULT now(),
            updated_at timestamp without time zone DEFAULT now()
        );

        CREATE SEQUENCE public.goadmin_roles_myid_seq
            START WITH 1
            INCREMENT BY 1
            NO MINVALUE
            MAXVALUE 99999999
            CACHE 1;

        CREATE TABLE public.goadmin_roles (
            id integer DEFAULT nextval('public.goadmin_roles_myid_seq'::regclass) NOT NULL,
            name character varying NOT NULL,
            slug character varying NOT NULL,
            created_at timestamp without time zone DEFAULT now(),
            updated_at timestamp without time zone DEFAULT now()
        );

        CREATE SEQUENCE public.goadmin_session_myid_seq
            START WITH 1
            INCREMENT BY 1
            NO MINVALUE
            MAXVALUE 99999999
            CACHE 1;

        CREATE TABLE public.goadmin_session (
            id integer DEFAULT nextval('public.goadmin_session_myid_seq'::regclass) NOT NULL,
            sid character varying(50) NOT NULL,
            "values" character varying(3000) NOT NULL,
            created_at timestamp without time zone DEFAULT now(),
            updated_at timestamp without time zone DEFAULT now()
        );

        CREATE SEQUENCE public.goadmin_site_myid_seq
            START WITH 1
            INCREMENT BY 1
            NO MINVALUE
            MAXVALUE 99999999
            CACHE 1;

        CREATE TABLE public.goadmin_site (
            id integer DEFAULT nextval('public.goadmin_site_myid_seq'::regclass) NOT NULL,
            key character varying(100) NOT NULL,
            value text NOT NULL,
            type integer DEFAULT 0,
            description character varying(3000),
            state integer DEFAULT 0,
            created_at timestamp without time zone DEFAULT now(),
            updated_at timestamp without time zone DEFAULT now()
        );

        CREATE TABLE public.goadmin_user_permissions (
            user_id integer NOT NULL,
            permission_id integer NOT NULL,
            created_at timestamp without time zone DEFAULT now(),
            updated_at timestamp without time zone DEFAULT now()
        );

        CREATE SEQUENCE public.goadmin_users_myid_seq
            START WITH 1
            INCREMENT BY 1
            NO MINVALUE
            MAXVALUE 99999999
            CACHE 1;

        CREATE TABLE public.goadmin_users (
            id integer DEFAULT nextval('public.goadmin_users_myid_seq'::regclass) NOT NULL,
            username character varying(100) NOT NULL,
            password character varying(100) NOT NULL,
            name character varying(100) NOT NULL,
            avatar character varying(255),
            remember_token character varying(100),
            created_at timestamp without time zone DEFAULT now(),
            updated_at timestamp without time zone DEFAULT now()
        );

        INSERT INTO public.goadmin_menu (id, parent_id, type, "order", title, header, plugin_name, icon, uri, uuid, created_at, updated_at) VALUES (1, 0, 1, 1, 'Dashboard', NULL, '', 'fa-bar-chart', '/', NULL, '2019-09-10 00:00:00', '2019-09-10 00:00:00');
        INSERT INTO public.goadmin_menu (id, parent_id, type, "order", title, header, plugin_name, icon, uri, uuid, created_at, updated_at) VALUES (2, 0, 1, 2, 'Admin', NULL, '', 'fa-tasks', '', NULL, '2019-09-10 00:00:00', '2019-09-10 00:00:00');
        INSERT INTO public.goadmin_menu (id, parent_id, type, "order", title, header, plugin_name, icon, uri, uuid, created_at, updated_at) VALUES (3, 2, 1, 2, 'Users', NULL, '', 'fa-users', '/info/manager', NULL, '2019-09-10 00:00:00', '2019-09-10 00:00:00');
        INSERT INTO public.goadmin_menu (id, parent_id, type, "order", title, header, plugin_name, icon, uri, uuid, created_at, updated_at) VALUES (4, 2, 1, 3, 'Roles', NULL, '', 'fa-user', '/info/roles', NULL, '2019-09-10 00:00:00', '2019-09-10 00:00:00');
        INSERT INTO public.goadmin_menu (id, parent_id, type, "order", title, header, plugin_name, icon, uri, uuid, created_at, updated_at) VALUES (5, 2, 1, 4, 'Permission', NULL, '', 'fa-ban', '/info/permission', NULL, '2019-09-10 00:00:00', '2019-09-10 00:00:00');
        INSERT INTO public.goadmin_menu (id, parent_id, type, "order", title, header, plugin_name, icon, uri, uuid, created_at, updated_at) VALUES (7, 2, 1, 6, 'Operation log', NULL, '', 'fa-history', '/info/op', NULL, '2019-09-10 00:00:00', '2019-09-10 00:00:00');
        INSERT INTO public.goadmin_menu (id, parent_id, type, "order", title, header, plugin_name, icon, uri, uuid, created_at, updated_at) VALUES (9, 0, 0, 9, 'Users', '', '', 'fa-users', '/info/users', NULL, '2019-09-10 00:00:00', '2019-09-10 00:00:00');
        INSERT INTO public.goadmin_menu (id, parent_id, type, "order", title, header, plugin_name, icon, uri, uuid, created_at, updated_at) VALUES (14, 0, 0, 12, 'Github', 'Miscellaneous', '', 'fa-github', 'https://github.com/shtorm-7/sing-box-extended', NULL, '2019-09-10 00:00:00', '2019-09-10 00:00:00');
        INSERT INTO public.goadmin_menu (id, parent_id, type, "order", title, header, plugin_name, icon, uri, uuid, created_at, updated_at) VALUES (15, 0, 0, 13, 'Donate', '', '', 'fa-heart', 'https://github.com/shtorm-7/sing-box-extended#support-the-project', NULL, '2019-09-10 00:00:00', '2019-09-10 00:00:00');
        INSERT INTO public.goadmin_menu (id, parent_id, type, "order", title, header, plugin_name, icon, uri, uuid, created_at, updated_at) VALUES (13, 0, 0, 7, 'Squads', 'General', '', 'fa-gg', '/info/squads', NULL, '2019-09-10 00:00:00', '2019-09-10 00:00:00');
        INSERT INTO public.goadmin_menu (id, parent_id, type, "order", title, header, plugin_name, icon, uri, uuid, created_at, updated_at) VALUES (11, 0, 0, 8, 'Nodes', '', '', 'fa-sitemap', '/info/nodes', NULL, '2019-09-10 00:00:00', '2019-09-10 00:00:00');
        INSERT INTO public.goadmin_menu (id, parent_id, type, "order", title, header, plugin_name, icon, uri, uuid, created_at, updated_at) VALUES (10, 0, 0, 10, 'Connection limiters', 'Limiters', '', 'fa-plug', '/info/connection_limiters', NULL, '2019-09-10 00:00:00', '2019-09-10 00:00:00');
        INSERT INTO public.goadmin_menu (id, parent_id, type, "order", title, header, plugin_name, icon, uri, uuid, created_at, updated_at) VALUES (8, 0, 0, 11, 'Bandwidth limiters', '', '', 'fa-dashboard', '/info/bandwidth_limiters', NULL, '2019-09-10 00:00:00', '2019-09-10 00:00:00');


        INSERT INTO public.goadmin_permissions (id, name, slug, http_method, http_path, created_at, updated_at) VALUES (1, 'All permission', '*', '', '*', '2019-09-10 00:00:00', '2019-09-10 00:00:00');
        INSERT INTO public.goadmin_permissions (id, name, slug, http_method, http_path, created_at, updated_at) VALUES (2, 'Dashboard', 'dashboard', 'GET,PUT,POST,DELETE', '/', '2019-09-10 00:00:00', '2019-09-10 00:00:00');


        INSERT INTO public.goadmin_role_menu (role_id, menu_id, created_at, updated_at) VALUES (1, 1, '2019-09-10 00:00:00', '2019-09-10 00:00:00');
        INSERT INTO public.goadmin_role_menu (role_id, menu_id, created_at, updated_at) VALUES (1, 7, '2019-09-10 00:00:00', '2019-09-10 00:00:00');
        INSERT INTO public.goadmin_role_menu (role_id, menu_id, created_at, updated_at) VALUES (2, 7, '2019-09-10 00:00:00', '2019-09-10 00:00:00');


        INSERT INTO public.goadmin_role_permissions (role_id, permission_id, created_at, updated_at) VALUES (1, 1, '2019-09-10 00:00:00', '2019-09-10 00:00:00');
        INSERT INTO public.goadmin_role_permissions (role_id, permission_id, created_at, updated_at) VALUES (1, 2, '2019-09-10 00:00:00', '2019-09-10 00:00:00');
        INSERT INTO public.goadmin_role_permissions (role_id, permission_id, created_at, updated_at) VALUES (2, 2, '2019-09-10 00:00:00', '2019-09-10 00:00:00');
        INSERT INTO public.goadmin_role_permissions (role_id, permission_id, created_at, updated_at) VALUES (0, 3, NULL, NULL);
        INSERT INTO public.goadmin_role_permissions (role_id, permission_id, created_at, updated_at) VALUES (0, 3, NULL, NULL);
        INSERT INTO public.goadmin_role_permissions (role_id, permission_id, created_at, updated_at) VALUES (0, 3, NULL, NULL);
        INSERT INTO public.goadmin_role_permissions (role_id, permission_id, created_at, updated_at) VALUES (0, 3, NULL, NULL);
        INSERT INTO public.goadmin_role_permissions (role_id, permission_id, created_at, updated_at) VALUES (0, 3, NULL, NULL);
        INSERT INTO public.goadmin_role_permissions (role_id, permission_id, created_at, updated_at) VALUES (0, 3, NULL, NULL);
        INSERT INTO public.goadmin_role_permissions (role_id, permission_id, created_at, updated_at) VALUES (0, 3, NULL, NULL);
        INSERT INTO public.goadmin_role_permissions (role_id, permission_id, created_at, updated_at) VALUES (0, 3, NULL, NULL);
        INSERT INTO public.goadmin_role_permissions (role_id, permission_id, created_at, updated_at) VALUES (0, 3, NULL, NULL);
        INSERT INTO public.goadmin_role_permissions (role_id, permission_id, created_at, updated_at) VALUES (0, 3, NULL, NULL);
        INSERT INTO public.goadmin_role_permissions (role_id, permission_id, created_at, updated_at) VALUES (0, 3, NULL, NULL);
        INSERT INTO public.goadmin_role_permissions (role_id, permission_id, created_at, updated_at) VALUES (0, 3, NULL, NULL);
        INSERT INTO public.goadmin_role_permissions (role_id, permission_id, created_at, updated_at) VALUES (0, 3, NULL, NULL);
        INSERT INTO public.goadmin_role_permissions (role_id, permission_id, created_at, updated_at) VALUES (0, 3, NULL, NULL);
        INSERT INTO public.goadmin_role_permissions (role_id, permission_id, created_at, updated_at) VALUES (0, 3, NULL, NULL);
        INSERT INTO public.goadmin_role_permissions (role_id, permission_id, created_at, updated_at) VALUES (0, 3, NULL, NULL);


        INSERT INTO public.goadmin_role_users (role_id, user_id, created_at, updated_at) VALUES (1, 1, '2019-09-10 00:00:00', '2019-09-10 00:00:00');
        INSERT INTO public.goadmin_role_users (role_id, user_id, created_at, updated_at) VALUES (2, 2, '2019-09-10 00:00:00', '2019-09-10 00:00:00');


        INSERT INTO public.goadmin_roles (id, name, slug, created_at, updated_at) VALUES (1, 'Administrator', 'administrator', '2019-09-10 00:00:00', '2019-09-10 00:00:00');
        INSERT INTO public.goadmin_roles (id, name, slug, created_at, updated_at) VALUES (2, 'Operator', 'operator', '2019-09-10 00:00:00', '2019-09-10 00:00:00');


        INSERT INTO public.goadmin_site (id, key, value, type, description, state, created_at, updated_at) VALUES (6, 'site_off', 'false', 0, NULL, 1, '2026-02-15 09:57:02.436501', '2026-02-15 09:57:02.436501');
        INSERT INTO public.goadmin_site (id, key, value, type, description, state, created_at, updated_at) VALUES (7, 'prohibit_config_modification', 'false', 0, NULL, 1, '2026-02-15 09:57:02.441183', '2026-02-15 09:57:02.441183');
        INSERT INTO public.goadmin_site (id, key, value, type, description, state, created_at, updated_at) VALUES (11, 'login_url', '/login', 0, NULL, 1, '2026-02-15 09:57:02.459525', '2026-02-15 09:57:02.459525');
        INSERT INTO public.goadmin_site (id, key, value, type, description, state, created_at, updated_at) VALUES (16, 'open_admin_api', 'false', 0, NULL, 1, '2026-02-15 09:57:02.483908', '2026-02-15 09:57:02.483908');
        INSERT INTO public.goadmin_site (id, key, value, type, description, state, created_at, updated_at) VALUES (18, 'domain', '', 0, NULL, 1, '2026-02-15 09:57:02.493151', '2026-02-15 09:57:02.493151');
        INSERT INTO public.goadmin_site (id, key, value, type, description, state, created_at, updated_at) VALUES (23, 'asset_root_path', './public/', 0, NULL, 1, '2026-02-15 09:57:02.517213', '2026-02-15 09:57:02.517213');
        INSERT INTO public.goadmin_site (id, key, value, type, description, state, created_at, updated_at) VALUES (24, 'url_prefix', 'admin', 0, NULL, 1, '2026-02-15 09:57:02.521815', '2026-02-15 09:57:02.521815');
        INSERT INTO public.goadmin_site (id, key, value, type, description, state, created_at, updated_at) VALUES (33, 'exclude_theme_components', 'null', 0, NULL, 1, '2026-02-15 09:57:02.565725', '2026-02-15 09:57:02.565725');
        INSERT INTO public.goadmin_site (id, key, value, type, description, state, created_at, updated_at) VALUES (39, 'app_id', 'Qn0eh7HQsrt9', 0, NULL, 1, '2026-02-15 09:57:02.592551', '2026-02-15 09:57:02.592551');
        INSERT INTO public.goadmin_site (id, key, value, type, description, state, created_at, updated_at) VALUES (41, 'auth_user_table', 'goadmin_users', 0, NULL, 1, '2026-02-15 09:57:02.601496', '2026-02-15 09:57:02.601496');
        INSERT INTO public.goadmin_site (id, key, value, type, description, state, created_at, updated_at) VALUES (53, 'bootstrap_file_path', '', 0, NULL, 1, '2026-02-15 09:57:02.658984', '2026-02-15 09:57:02.658984');
        INSERT INTO public.goadmin_site (id, key, value, type, description, state, created_at, updated_at) VALUES (55, 'index_url', '/', 0, NULL, 1, '2026-02-15 09:57:02.668457', '2026-02-15 09:57:02.668457');
        INSERT INTO public.goadmin_site (id, key, value, type, description, state, created_at, updated_at) VALUES (66, 'login_logo', '', 0, NULL, 1, '2026-02-15 09:57:02.719608', '2026-02-15 09:57:02.719608');
        INSERT INTO public.goadmin_site (id, key, value, type, description, state, created_at, updated_at) VALUES (67, 'hide_visitor_user_center_entrance', 'false', 0, NULL, 1, '2026-02-15 09:57:02.724307', '2026-02-15 09:57:02.724307');
        INSERT INTO public.goadmin_site (id, key, value, type, description, state, created_at, updated_at) VALUES (68, 'go_mod_file_path', '', 0, NULL, 1, '2026-02-15 09:57:02.728694', '2026-02-15 09:57:02.728694');
        INSERT INTO public.goadmin_site (id, key, value, type, description, state, created_at, updated_at) VALUES (3, 'logger_encoder_caller', 'full', 0, NULL, 1, '2026-02-15 09:57:02.420312', '2026-02-15 09:57:02.420312');
        INSERT INTO public.goadmin_site (id, key, value, type, description, state, created_at, updated_at) VALUES (60, 'logger_encoder_caller_key', 'caller', 0, NULL, 1, '2026-02-15 09:57:02.692189', '2026-02-15 09:57:02.692189');
        INSERT INTO public.goadmin_site (id, key, value, type, description, state, created_at, updated_at) VALUES (34, 'logo', 'Sing-box Extended', 0, NULL, 1, '2026-02-15 09:57:02.570594', '2026-02-15 09:57:02.570594');
        INSERT INTO public.goadmin_site (id, key, value, type, description, state, created_at, updated_at) VALUES (69, 'env', 'prod', 0, NULL, 1, '2026-02-15 09:57:02.733059', '2026-02-15 09:57:02.733059');
        INSERT INTO public.goadmin_site (id, key, value, type, description, state, created_at, updated_at) VALUES (29, 'color_scheme', 'skin-black', 0, NULL, 1, '2026-02-15 09:57:02.545599', '2026-02-15 09:57:02.545599');
        INSERT INTO public.goadmin_site (id, key, value, type, description, state, created_at, updated_at) VALUES (17, 'allow_del_operation_log', 'false', 0, NULL, 1, '2026-02-15 09:57:02.488458', '2026-02-15 09:57:02.488458');
        INSERT INTO public.goadmin_site (id, key, value, type, description, state, created_at, updated_at) VALUES (35, 'info_log_path', '', 0, NULL, 1, '2026-02-15 09:57:02.574649', '2026-02-15 09:57:02.574649');
        INSERT INTO public.goadmin_site (id, key, value, type, description, state, created_at, updated_at) VALUES (22, 'operation_log_off', 'false', 0, NULL, 1, '2026-02-15 09:57:02.512394', '2026-02-15 09:57:02.512394');
        INSERT INTO public.goadmin_site (id, key, value, type, description, state, created_at, updated_at) VALUES (42, 'hide_app_info_entrance', 'false', 0, NULL, 1, '2026-02-15 09:57:02.606071', '2026-02-15 09:57:02.606071');
        INSERT INTO public.goadmin_site (id, key, value, type, description, state, created_at, updated_at) VALUES (12, 'access_log_path', '', 0, NULL, 1, '2026-02-15 09:57:02.464612', '2026-02-15 09:57:02.464612');
        INSERT INTO public.goadmin_site (id, key, value, type, description, state, created_at, updated_at) VALUES (32, 'logger_rotate_max_age', '30', 0, NULL, 1, '2026-02-15 09:57:02.560801', '2026-02-15 09:57:02.560801');
        INSERT INTO public.goadmin_site (id, key, value, type, description, state, created_at, updated_at) VALUES (40, 'custom_foot_html', '', 0, NULL, 1, '2026-02-15 09:57:02.597285', '2026-02-15 09:57:02.597285');
        INSERT INTO public.goadmin_site (id, key, value, type, description, state, created_at, updated_at) VALUES (62, 'logger_encoder_duration', 'string', 0, NULL, 1, '2026-02-15 09:57:02.701522', '2026-02-15 09:57:02.701522');
        INSERT INTO public.goadmin_site (id, key, value, type, description, state, created_at, updated_at) VALUES (65, 'logger_encoder_level_key', 'level', 0, NULL, 1, '2026-02-15 09:57:02.715108', '2026-02-15 09:57:02.715108');
        INSERT INTO public.goadmin_site (id, key, value, type, description, state, created_at, updated_at) VALUES (64, 'debug', 'false', 0, NULL, 1, '2026-02-15 09:57:02.710705', '2026-02-15 09:57:02.710705');
        INSERT INTO public.goadmin_site (id, key, value, type, description, state, created_at, updated_at) VALUES (43, 'hide_plugin_entrance', 'false', 0, NULL, 1, '2026-02-15 09:57:02.610825', '2026-02-15 09:57:02.610825');
        INSERT INTO public.goadmin_site (id, key, value, type, description, state, created_at, updated_at) VALUES (54, 'animation_type', '', 0, NULL, 1, '2026-02-15 09:57:02.663713', '2026-02-15 09:57:02.663713');
        INSERT INTO public.goadmin_site (id, key, value, type, description, state, created_at, updated_at) VALUES (48, 'theme', 'sword', 0, NULL, 1, '2026-02-15 09:57:02.634039', '2026-02-15 09:57:02.634039');
        INSERT INTO public.goadmin_site (id, key, value, type, description, state, created_at, updated_at) VALUES (45, 'info_log_off', 'false', 0, NULL, 1, '2026-02-15 09:57:02.620165', '2026-02-15 09:57:02.620165');
        INSERT INTO public.goadmin_site (id, key, value, type, description, state, created_at, updated_at) VALUES (31, 'error_log_path', '', 0, NULL, 1, '2026-02-15 09:57:02.555798', '2026-02-15 09:57:02.555798');
        INSERT INTO public.goadmin_site (id, key, value, type, description, state, created_at, updated_at) VALUES (5, 'asset_url', '', 0, NULL, 1, '2026-02-15 09:57:02.431855', '2026-02-15 09:57:02.431855');
        INSERT INTO public.goadmin_site (id, key, value, type, description, state, created_at, updated_at) VALUES (36, 'logger_encoder_encoding', 'console', 0, NULL, 1, '2026-02-15 09:57:02.579052', '2026-02-15 09:57:02.579052');
        INSERT INTO public.goadmin_site (id, key, value, type, description, state, created_at, updated_at) VALUES (27, 'login_title', 'Sing-box Extended', 0, NULL, 1, '2026-02-15 09:57:02.536102', '2026-02-15 09:57:02.536102');
        INSERT INTO public.goadmin_site (id, key, value, type, description, state, created_at, updated_at) VALUES (51, 'animation_duration', '0.00', 0, NULL, 1, '2026-02-15 09:57:02.64867', '2026-02-15 09:57:02.64867');
        INSERT INTO public.goadmin_site (id, key, value, type, description, state, created_at, updated_at) VALUES (19, 'file_upload_engine', '{"name":"local"}', 0, NULL, 1, '2026-02-15 09:57:02.49794', '2026-02-15 09:57:02.49794');
        INSERT INTO public.goadmin_site (id, key, value, type, description, state, created_at, updated_at) VALUES (26, 'logger_encoder_time', 'iso8601', 0, NULL, 1, '2026-02-15 09:57:02.531365', '2026-02-15 09:57:02.531365');
        INSERT INTO public.goadmin_site (id, key, value, type, description, state, created_at, updated_at) VALUES (10, 'custom_404_html', '', 0, NULL, 1, '2026-02-15 09:57:02.454777', '2026-02-15 09:57:02.454777');
        INSERT INTO public.goadmin_site (id, key, value, type, description, state, created_at, updated_at) VALUES (58, 'sql_log', 'false', 0, NULL, 1, '2026-02-15 09:57:02.682567', '2026-02-15 09:57:02.682567');
        INSERT INTO public.goadmin_site (id, key, value, type, description, state, created_at, updated_at) VALUES (2, 'logger_encoder_message_key', 'msg', 0, NULL, 1, '2026-02-15 09:57:02.415189', '2026-02-15 09:57:02.415189');
        INSERT INTO public.goadmin_site (id, key, value, type, description, state, created_at, updated_at) VALUES (46, 'logger_encoder_stacktrace_key', 'stacktrace', 0, NULL, 1, '2026-02-15 09:57:02.624977', '2026-02-15 09:57:02.624977');
        INSERT INTO public.goadmin_site (id, key, value, type, description, state, created_at, updated_at) VALUES (63, 'mini_logo', 'SBE', 0, NULL, 1, '2026-02-15 09:57:02.706145', '2026-02-15 09:57:02.706145');
        INSERT INTO public.goadmin_site (id, key, value, type, description, state, created_at, updated_at) VALUES (38, 'custom_403_html', '', 0, NULL, 1, '2026-02-15 09:57:02.588062', '2026-02-15 09:57:02.588062');
        INSERT INTO public.goadmin_site (id, key, value, type, description, state, created_at, updated_at) VALUES (30, 'language', 'en', 0, NULL, 1, '2026-02-15 09:57:02.550466', '2026-02-15 09:57:02.550466');
        INSERT INTO public.goadmin_site (id, key, value, type, description, state, created_at, updated_at) VALUES (15, 'hide_config_center_entrance', 'false', 0, NULL, 1, '2026-02-15 09:57:02.479097', '2026-02-15 09:57:02.479097');
        INSERT INTO public.goadmin_site (id, key, value, type, description, state, created_at, updated_at) VALUES (59, 'logger_rotate_max_backups', '5', 0, NULL, 1, '2026-02-15 09:57:02.687429', '2026-02-15 09:57:02.687429');
        INSERT INTO public.goadmin_site (id, key, value, type, description, state, created_at, updated_at) VALUES (57, 'custom_head_html', '', 0, NULL, 1, '2026-02-15 09:57:02.677723', '2026-02-15 09:57:02.677723');
        INSERT INTO public.goadmin_site (id, key, value, type, description, state, created_at, updated_at) VALUES (52, 'custom_500_html', '', 0, NULL, 1, '2026-02-15 09:57:02.654236', '2026-02-15 09:57:02.654236');
        INSERT INTO public.goadmin_site (id, key, value, type, description, state, created_at, updated_at) VALUES (44, 'title', 'Sing-box Extended', 0, NULL, 1, '2026-02-15 09:57:02.615471', '2026-02-15 09:57:02.615471');
        INSERT INTO public.goadmin_site (id, key, value, type, description, state, created_at, updated_at) VALUES (47, 'session_life_time', '7200', 0, NULL, 1, '2026-02-15 09:57:02.629619', '2026-02-15 09:57:02.629619');
        INSERT INTO public.goadmin_site (id, key, value, type, description, state, created_at, updated_at) VALUES (8, 'access_log_off', 'false', 0, NULL, 1, '2026-02-15 09:57:02.445593', '2026-02-15 09:57:02.445593');
        INSERT INTO public.goadmin_site (id, key, value, type, description, state, created_at, updated_at) VALUES (49, 'error_log_off', 'false', 0, NULL, 1, '2026-02-15 09:57:02.6385', '2026-02-15 09:57:02.6385');
        INSERT INTO public.goadmin_site (id, key, value, type, description, state, created_at, updated_at) VALUES (50, 'logger_rotate_max_size', '10', 0, NULL, 1, '2026-02-15 09:57:02.643733', '2026-02-15 09:57:02.643733');
        INSERT INTO public.goadmin_site (id, key, value, type, description, state, created_at, updated_at) VALUES (14, 'logger_rotate_compress', 'false', 0, NULL, 1, '2026-02-15 09:57:02.474296', '2026-02-15 09:57:02.474296');
        INSERT INTO public.goadmin_site (id, key, value, type, description, state, created_at, updated_at) VALUES (13, 'logger_encoder_time_key', 'ts', 0, NULL, 1, '2026-02-15 09:57:02.469396', '2026-02-15 09:57:02.469396');
        INSERT INTO public.goadmin_site (id, key, value, type, description, state, created_at, updated_at) VALUES (37, 'animation_delay', '0.00', 0, NULL, 1, '2026-02-15 09:57:02.583815', '2026-02-15 09:57:02.583815');
        INSERT INTO public.goadmin_site (id, key, value, type, description, state, created_at, updated_at) VALUES (20, 'extra', '', 0, NULL, 1, '2026-02-15 09:57:02.50276', '2026-02-15 09:57:02.50276');
        INSERT INTO public.goadmin_site (id, key, value, type, description, state, created_at, updated_at) VALUES (25, 'access_assets_log_off', 'false', 0, NULL, 1, '2026-02-15 09:57:02.526618', '2026-02-15 09:57:02.526618');
        INSERT INTO public.goadmin_site (id, key, value, type, description, state, created_at, updated_at) VALUES (4, 'logger_level', '0', 0, NULL, 1, '2026-02-15 09:57:02.426736', '2026-02-15 09:57:02.426736');
        INSERT INTO public.goadmin_site (id, key, value, type, description, state, created_at, updated_at) VALUES (9, 'footer_info', '', 0, NULL, 1, '2026-02-15 09:57:02.450409', '2026-02-15 09:57:02.450409');
        INSERT INTO public.goadmin_site (id, key, value, type, description, state, created_at, updated_at) VALUES (21, 'no_limit_login_ip', 'false', 0, NULL, 1, '2026-02-15 09:57:02.507609', '2026-02-15 09:57:02.507609');
        INSERT INTO public.goadmin_site (id, key, value, type, description, state, created_at, updated_at) VALUES (28, 'hide_tool_entrance', 'false', 0, NULL, 1, '2026-02-15 09:57:02.540813', '2026-02-15 09:57:02.540813');
        INSERT INTO public.goadmin_site (id, key, value, type, description, state, created_at, updated_at) VALUES (61, 'logger_encoder_level', 'capitalColor', 0, NULL, 1, '2026-02-15 09:57:02.696859', '2026-02-15 09:57:02.696859');
        INSERT INTO public.goadmin_site (id, key, value, type, description, state, created_at, updated_at) VALUES (56, 'logger_encoder_name_key', 'logger', 0, NULL, 1, '2026-02-15 09:57:02.672962', '2026-02-15 09:57:02.672962');


        INSERT INTO public.goadmin_user_permissions (user_id, permission_id, created_at, updated_at) VALUES (1, 1, '2019-09-10 00:00:00', '2019-09-10 00:00:00');
        INSERT INTO public.goadmin_user_permissions (user_id, permission_id, created_at, updated_at) VALUES (2, 2, '2019-09-10 00:00:00', '2019-09-10 00:00:00');
        INSERT INTO public.goadmin_user_permissions (user_id, permission_id, created_at, updated_at) VALUES (0, 1, NULL, NULL);
        INSERT INTO public.goadmin_user_permissions (user_id, permission_id, created_at, updated_at) VALUES (0, 1, NULL, NULL);
        INSERT INTO public.goadmin_user_permissions (user_id, permission_id, created_at, updated_at) VALUES (0, 1, NULL, NULL);
        INSERT INTO public.goadmin_user_permissions (user_id, permission_id, created_at, updated_at) VALUES (0, 1, NULL, NULL);
        INSERT INTO public.goadmin_user_permissions (user_id, permission_id, created_at, updated_at) VALUES (0, 1, NULL, NULL);
        INSERT INTO public.goadmin_user_permissions (user_id, permission_id, created_at, updated_at) VALUES (0, 1, NULL, NULL);
        INSERT INTO public.goadmin_user_permissions (user_id, permission_id, created_at, updated_at) VALUES (0, 1, NULL, NULL);
        INSERT INTO public.goadmin_user_permissions (user_id, permission_id, created_at, updated_at) VALUES (0, 1, NULL, NULL);
        INSERT INTO public.goadmin_user_permissions (user_id, permission_id, created_at, updated_at) VALUES (0, 1, NULL, NULL);
        INSERT INTO public.goadmin_user_permissions (user_id, permission_id, created_at, updated_at) VALUES (0, 1, NULL, NULL);
        INSERT INTO public.goadmin_user_permissions (user_id, permission_id, created_at, updated_at) VALUES (0, 1, NULL, NULL);
        INSERT INTO public.goadmin_user_permissions (user_id, permission_id, created_at, updated_at) VALUES (0, 1, NULL, NULL);
        INSERT INTO public.goadmin_user_permissions (user_id, permission_id, created_at, updated_at) VALUES (0, 1, NULL, NULL);
        INSERT INTO public.goadmin_user_permissions (user_id, permission_id, created_at, updated_at) VALUES (0, 1, NULL, NULL);
        INSERT INTO public.goadmin_user_permissions (user_id, permission_id, created_at, updated_at) VALUES (0, 1, NULL, NULL);
        INSERT INTO public.goadmin_user_permissions (user_id, permission_id, created_at, updated_at) VALUES (0, 1, NULL, NULL);


        INSERT INTO public.goadmin_users (id, username, password, name, avatar, remember_token, created_at, updated_at) VALUES (2, 'operator', '$2a$10$rVqkOzHjN2MdlEprRflb1eGP0oZXuSrbJLOmJagFsCd81YZm0bsh.', 'Operator', '', NULL, '2019-09-10 00:00:00', '2019-09-10 00:00:00');
        INSERT INTO public.goadmin_users (id, username, password, name, avatar, remember_token, created_at, updated_at) VALUES (1, 'admin', '$2a$10$ilNHHnX5S6EMw.Ffc1Y1JezYCyquFIO.7Z0vLr1eHJUXnGy4cdrtq', 'admin', '', 'tlNcBVK9AvfYH7WEnwB1RKvocJu8FfRy4um3DJtwdHuJy0dwFsLOgAc0xUfh', '2019-09-10 00:00:00', '2019-09-10 00:00:00');


        SELECT pg_catalog.setval('public.goadmin_menu_myid_seq', 12, true);


        SELECT pg_catalog.setval('public.goadmin_operation_log_myid_seq', 11, true);


        SELECT pg_catalog.setval('public.goadmin_permissions_myid_seq', 2, true);


        SELECT pg_catalog.setval('public.goadmin_roles_myid_seq', 2, true);


        SELECT pg_catalog.setval('public.goadmin_session_myid_seq', 7, true);


        SELECT pg_catalog.setval('public.goadmin_site_myid_seq', 69, true);


        SELECT pg_catalog.setval('public.goadmin_users_myid_seq', 2, true);


        ALTER TABLE ONLY public.goadmin_menu
            ADD CONSTRAINT goadmin_menu_pkey PRIMARY KEY (id);


        ALTER TABLE ONLY public.goadmin_operation_log
            ADD CONSTRAINT goadmin_operation_log_pkey PRIMARY KEY (id);


        ALTER TABLE ONLY public.goadmin_permissions
            ADD CONSTRAINT goadmin_permissions_pkey PRIMARY KEY (id);


        ALTER TABLE ONLY public.goadmin_roles
            ADD CONSTRAINT goadmin_roles_pkey PRIMARY KEY (id);


        ALTER TABLE ONLY public.goadmin_session
            ADD CONSTRAINT goadmin_session_pkey PRIMARY KEY (id);


        ALTER TABLE ONLY public.goadmin_site
            ADD CONSTRAINT goadmin_site_pkey PRIMARY KEY (id);


        ALTER TABLE ONLY public.goadmin_users
            ADD CONSTRAINT goadmin_users_pkey PRIMARY KEY (id);
    `,
	"1_initialize_schema.down.sql": ``,
}

func MigratePostgreSQL(db *sql.DB) error {
	driver, err := postgres.WithInstance(db, &postgres.Config{})
	if err != nil {
		return err
	}

	sourceDriver := source.NewRawDriver(migrations)
	if err := sourceDriver.Init(); err != nil {
		return err
	}

	m, err := migrate.NewWithInstance(
		"raw",
		sourceDriver,
		"postgres",
		driver,
	)
	if err != nil {
		return err
	}

	return m.Up()
}
