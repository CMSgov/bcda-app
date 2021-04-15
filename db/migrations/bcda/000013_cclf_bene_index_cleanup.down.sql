BEGIN;

CREATE INDEX IF NOT EXISTS idx_cclf_beneficiaries_bb_id ON public.cclf_beneficiaries USING btree (blue_button_id);
CREATE INDEX IF NOT EXISTS idx_cclf_beneficiaries_mbi ON public.cclf_beneficiaries USING btree (mbi);

COMMIT;