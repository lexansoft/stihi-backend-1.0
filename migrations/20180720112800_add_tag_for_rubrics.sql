ALTER TABLE rubrics
    ADD COLUMN tag_name VARCHAR(255);

UPDATE rubrics SET tag_name = 'Сонеты канцоны рондо' WHERE id = 230;
UPDATE rubrics SET tag_name = 'Рубаи танка хокку', name = 'Рубаи, танка, хокку' WHERE id = 240;
