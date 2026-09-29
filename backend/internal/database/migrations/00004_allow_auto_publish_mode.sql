-- +goose Up
update articles
set publish_mode = 'auto'
where publish_mode = 'automatic';

alter table articles
    drop constraint if exists articles_publish_mode_check;

alter table articles
    add constraint articles_publish_mode_check
    check (publish_mode in ('manual', 'auto'));

-- +goose Down
update articles
set publish_mode = 'automatic'
where publish_mode = 'auto';

alter table articles
    drop constraint if exists articles_publish_mode_check;

alter table articles
    add constraint articles_publish_mode_check
    check (publish_mode in ('manual', 'automatic'));
