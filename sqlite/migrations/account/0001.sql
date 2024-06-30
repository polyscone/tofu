create table account__users (
	id                          text not null primary key,
	email                       text not null unique collate nocase,
	hashed_password             blob,
	totp_method                 text not null,
	totp_tel                    text not null,
	totp_key                    blob,
	totp_algorithm              text not null,
	totp_digits                 integer not null,
	totp_period                 text not null,
	totp_verified_at            text,
	totp_activated_at           text,
	invited_at                  text,
	signed_up_at                text,
	signed_up_system            text not null,
	signed_up_method            text not null,
	verified_at                 text,
	activated_at                text,
	last_sign_in_attempt_at     text,
	last_sign_in_attempt_system text not null,
	last_sign_in_attempt_method text not null,
	last_signed_in_at           text,
	last_signed_in_system       text not null,
	last_signed_in_method       text not null,
	suspended_at                text,
	suspended_reason            text not null,
	created_at                  text not null,
	updated_at                  text
) strict;

create table account__totp_reset_requests (
	user_id      text not null primary key references account__users(id) on delete cascade on update cascade,
	requested_at text,
	approved_at  text,
	created_at   text not null,
	updated_at   text
) strict;

create table account__sign_in_attempt_logs (
	email           text not null primary key,
	attempts        integer not null,
	last_attempt_at text not null,
	created_at      text not null,
	updated_at      text
) strict;

create table account__roles (
	id          text not null primary key,
	name        text not null unique collate nocase,
	description text not null collate nocase,
	created_at  text not null,
	updated_at  text
) strict;

create table account__permissions (
	id         text not null primary key,
	name       text not null unique collate nocase,
	created_at text not null,
	updated_at text
) strict;

create table account__role_permissions (
	role_id       text not null references account__roles(id) on delete cascade on update cascade,
	permission_id text not null references account__permissions(id) on delete cascade on update cascade,
	created_at    text not null,
	updated_at    text,
	primary key (role_id, permission_id)
) strict;

create table account__user_roles (
	user_id    text not null references account__users(id) on delete cascade on update cascade,
	role_id    text not null references account__roles(id) on delete cascade on update cascade,
	created_at text not null,
	updated_at text,
	primary key (user_id, role_id)
) strict;

create table account__user_grants (
	user_id       text not null references account__users(id) on delete cascade on update cascade,
	permission_id text not null references account__permissions(id) on delete cascade on update cascade,
	created_at    text not null,
	updated_at    text,
	primary key (user_id, permission_id)
) strict;

create table account__user_denials (
	user_id       text not null references account__users(id) on delete cascade on update cascade,
	permission_id text not null references account__permissions(id) on delete cascade on update cascade,
	created_at    text not null,
	updated_at    text,
	primary key (user_id, permission_id)
) strict;

create table account__recovery_codes (
	user_id     text not null references account__users(id) on delete cascade on update cascade,
	hashed_code blob not null,
	created_at  text not null,
	updated_at  text,
	primary key (user_id, hashed_code)
) strict;
