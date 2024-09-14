CREATE TABLE grades_org( id serial NOT NULL, g INT NOT NULL );
INSERT INTO grades_org(g)SELECT FLOOR(random()*100) FROM generate_series(0, 10000000);
CREATE INDEX grades_org_index ON grades_org(g);
