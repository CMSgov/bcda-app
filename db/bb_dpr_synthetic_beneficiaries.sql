/* 
 * This script duplicates the existing data in the beneficiaries table, but adds a '-' (negative sign) in front of each patient_id.
 * The intent is for this script to be executed on a DB that contains synthetic data and will be used with the BB DPR.
 */
insert into beneficiaries (patient_id, aco_id) (select concat('-', patient_id) as patient_id, aco_id as aco_id from beneficiaries);
