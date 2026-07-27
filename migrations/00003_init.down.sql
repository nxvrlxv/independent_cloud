ALTER TABLE folders DROP CONSTRAINT folders_parent_id_fkey;
ALTER TABLE folders ADD CONSTRAINT folders_parent_id_fkey FOREIGN KEY (parent_id) REFERENCES folders(folder_id);