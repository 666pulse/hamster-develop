create table t_icp_canister_controller
(
    id            int auto_increment
        primary key,
    fk_user_id    int                                 null comment 'adder',
    canister_id   varchar(50)                         null,
    controller    varchar(100)                        null comment '控制者',
    create_time   timestamp default CURRENT_TIMESTAMP null comment '创建时间'
)