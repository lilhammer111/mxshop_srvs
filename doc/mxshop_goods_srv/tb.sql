-- auto-generated definition
create table banner
(
    id          int auto_increment
        primary key,
    add_time    datetime(3)   null,
    update_time datetime(3)   null,
    deleted_at  datetime(3)   null,
    is_deleted  tinyint(1)    null,
    image       varchar(200)  not null,
    url         varchar(200)  not null,
    `index`     int default 1 not null
);

-- auto-generated definition
create table brand
(
    id          int auto_increment
        primary key,
    add_time    datetime(3)            null,
    update_time datetime(3)            null,
    deleted_at  datetime(3)            null,
    is_deleted  tinyint(1)             null,
    name        varchar(20)            not null,
    logo        varchar(20) default '' not null
);

-- auto-generated definition
create table category
(
    id                 int auto_increment
        primary key,
    add_time           datetime(3)          null,
    update_time        datetime(3)          null,
    deleted_at         datetime(3)          null,
    is_deleted         tinyint(1)           null,
    name               varchar(20)          not null,
    parent_category_id int                  null,
    level              int        default 1 not null,
    is_tab             tinyint(1) default 0 not null,
    constraint fk_category_sub_category
        foreign key (parent_category_id) references category (id)
);

-- auto-generated definition
create table good
(
    id            int auto_increment
        primary key,
    add_time      datetime(3)          null,
    update_time   datetime(3)          null,
    deleted_at    datetime(3)          null,
    is_deleted    tinyint(1)           null,
    category_id   int                  not null,
    brand_id      int                  not null,
    on_sale       tinyint(1) default 0 not null,
    ship_free     tinyint(1) default 0 not null,
    is_new        tinyint(1) default 0 not null,
    is_hot        tinyint(1) default 0 not null,
    name          varchar(50)          not null,
    goods_sn      varchar(50)          not null,
    hit_num       int        default 0 not null,
    sales_volumes int        default 0 not null,
    favorites     int        default 0 not null,
    market_price  float                not null,
    shop_price    float                not null,
    brief         varchar(50)          not null,
    images        varchar(1000)        not null,
    desc_images   varchar(1000)        not null,
    cover_image   varchar(200)         not null,
    constraint fk_good_brand
        foreign key (brand_id) references brand (id),
    constraint fk_good_category
        foreign key (category_id) references category (id)
);

-- auto-generated definition
create table goods_category_brand
(
    id          int auto_increment
        primary key,
    add_time    datetime(3) null,
    update_time datetime(3) null,
    deleted_at  datetime(3) null,
    is_deleted  tinyint(1)  null,
    category_id int         null,
    brand_id    int         null,
    constraint idx_category_brand
        unique (category_id, brand_id),
    constraint fk_goods_category_brand_brand
        foreign key (brand_id) references brand (id),
    constraint fk_goods_category_brand_category
        foreign key (category_id) references category (id)
);

