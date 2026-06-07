create table `project__slack_notification` (
    `id` integer primary key autoincrement,
    `project_id` int not null,
    `name` varchar(255) not null,
    `description` text null,
    `token` varchar(255) not null,
    `channel` varchar(255) not null,
    `color` varchar(7) not null default '',
    `started_message` text null,
    `success_message` text null,
    `error_message` text null,
    foreign key (`project_id`) references project(`id`) on delete cascade
);

alter table `project__template` add `suppress_failure_alerts` bool not null default false;
alter table `project__template` add `slack_notification_started_id` int null references project__slack_notification(`id`) on delete set null;
alter table `project__template` add `slack_notification_success_id` int null references project__slack_notification(`id`) on delete set null;
alter table `project__template` add `slack_notification_failure_id` int null references project__slack_notification(`id`) on delete set null;
