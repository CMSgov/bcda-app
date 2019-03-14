ALTER TABLE beneficiaries
    DROP CONSTRAINT beneficiaries_pkey,
    DROP CONSTRAINT beneficiaries_aco_id_fkey,
    DROP COLUMN beneficiary_id,
    DROP COLUMN patient_id,
    DROP COLUMN aco_id,
    ADD PRIMARY KEY (id);

DELETE FROM beneficiaries WHERE blue_button_id IS null;