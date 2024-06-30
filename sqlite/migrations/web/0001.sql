create table web__sessions (
	id         text not null primary key,
	data       blob not null,
	created_at text not null,
	updated_at text not null
) strict;

create table web__tokens (
	hash       blob not null primary key,
	value      text not null,
	kind       text not null,
	expires_at text not null,
	created_at text not null,
	updated_at text
) strict;
