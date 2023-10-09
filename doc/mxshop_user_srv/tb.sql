-- auto-generated definition
create table user
(
    id          int auto_increment
        primary key,
    add_time    datetime(3)               null,
    update_time datetime(3)               null,
    deleted_at  datetime(3)               null,
    is_deleted  tinyint(1)                null,
    mobile      varchar(11)               not null,
    password    varchar(100)              not null,
    nick_name   varchar(20)               null,
    birthday    datetime                  null,
    gender      varchar(6) default 'male' null comment 'female表示女, male表示男',
    role        int        default 1      null comment '1表示普通用户， 2表示管理员',
    constraint mobile
        unique (mobile)
);

create index idx_mobile
    on user (mobile);