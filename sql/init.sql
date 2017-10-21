
DROP DATABASE IF EXISTS doc_colab;
CREATE DATABASE doc_colab;
\connect doc_colab;

DROP TABLE IF EXISTS documnets;
CREATE TABLE documents (
    id SERIAL PRIMARY KEY NOT NULL UNIQUE,
    title TEXT NOT NULL,
     -- slug VARCHAR(255) NOT NULL UNIQUE,
    created TIMESTAMP WITHOUT TIME ZONE DEFAULT (NOW() AT TIME ZONE 'UTC')
);

DROP TABLE IF EXISTS document_changes;
CREATE TABLE document_changes (
    document_id INT NOT NULL,
    body_state TEXT,
    updated TIMESTAMP WITHOUT TIME ZONE
);