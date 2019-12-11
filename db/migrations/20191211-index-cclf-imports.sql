CREATE INDEX idx_cclf_beneficiaries_file_id ON public.cclf_beneficiaries USING btree (file_id);
CREATE INDEX idx_cclf_beneficiaries_hicn ON public.cclf_beneficiaries USING btree (hicn);
CREATE INDEX idx_cclf_beneficiaries_mbi ON public.cclf_beneficiaries USING btree (mbi);