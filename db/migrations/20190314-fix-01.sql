CREATE TABLE acos_beneficiaries (
  aco_id uuid NOT NULL,
  beneficiary_id int NOT NULL
);

INSERT INTO acos_beneficiaries (aco_id, beneficiary_id) 
(SELECT aco_id, beneficiary_id FROM beneficiaries);

ALTER TABLE beneficiaries
    RENAME COLUMN beneficiary_id TO id;
  
ALTER TABLE beneficiaries
    RENAME COLUMN patient_id TO blue_button_id;

ALTER TABLE beneficiaries
    DROP CONSTRAINT beneficiaries_aco_id_fkey,
    DROP COLUMN aco_id;
