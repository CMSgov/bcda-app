-- Script migrates our cclf_beneficiaries id column from an int to bigint.
-- It does this by the following steps:
-- 1. creating a temporary table that has id set to bigint
-- 2. copying over all the rows from the original table in batches
-- 3. duplicating the primary key, foreign key and index constraints (we opted to not add in the index on mbi and blue_button_id)
-- 4. adding the defaults back into the id and bookkeeping columns
-- 5. renaming the original table to `_old` and renaming the temp table to `cclf_beneficiaries` so it is a drop in replacement
-- 6. change ownership of the cclf id sequence to the new table
-- 7. setup the trigger for the timestamp on the updated_at row
-- 8. drop the old table
-- 9. finally, rename the constraints in the new table to the original names
CREATE TABLE IF NOT EXISTS public.cclf_beneficiaries_temp (
    id bigint NOT NULL,
    created_at timestamp with time zone,
    updated_at timestamp with time zone,
    file_id integer NOT NULL,
    mbi character(11) NOT NULL,
    blue_button_id text
);

INSERT INTO cclf_beneficiaries_temp (SELECT * FROM cclf_beneficiaries WHERE id between 67606581 and 67999999);
INSERT INTO cclf_beneficiaries_temp (SELECT * FROM cclf_beneficiaries WHERE id between 68000000 and 68999999);
INSERT INTO cclf_beneficiaries_temp (SELECT * FROM cclf_beneficiaries WHERE id between 69000000 and 69999999);
INSERT INTO cclf_beneficiaries_temp (SELECT * FROM cclf_beneficiaries WHERE id between 70000000 and 70999999);
INSERT INTO cclf_beneficiaries_temp (SELECT * FROM cclf_beneficiaries WHERE id between 71000000 and 71999999);
INSERT INTO cclf_beneficiaries_temp (SELECT * FROM cclf_beneficiaries WHERE id between 72000000 and 72999999);
INSERT INTO cclf_beneficiaries_temp (SELECT * FROM cclf_beneficiaries WHERE id between 73000000 and 73999999);
INSERT INTO cclf_beneficiaries_temp (SELECT * FROM cclf_beneficiaries WHERE id between 74000000 and 74999999);
INSERT INTO cclf_beneficiaries_temp (SELECT * FROM cclf_beneficiaries WHERE id between 75000000 and 75999999);
INSERT INTO cclf_beneficiaries_temp (SELECT * FROM cclf_beneficiaries WHERE id between 76000000 and 76999999);
INSERT INTO cclf_beneficiaries_temp (SELECT * FROM cclf_beneficiaries WHERE id between 77000000 and 77999999);
INSERT INTO cclf_beneficiaries_temp (SELECT * FROM cclf_beneficiaries WHERE id between 78000000 and 78999999);
INSERT INTO cclf_beneficiaries_temp (SELECT * FROM cclf_beneficiaries WHERE id between 79000000 and 79999999);
INSERT INTO cclf_beneficiaries_temp (SELECT * FROM cclf_beneficiaries WHERE id between 80000000 and 80999999);
INSERT INTO cclf_beneficiaries_temp (SELECT * FROM cclf_beneficiaries WHERE id between 81000000 and 81999999);
INSERT INTO cclf_beneficiaries_temp (SELECT * FROM cclf_beneficiaries WHERE id between 82000000 and 82999999);
INSERT INTO cclf_beneficiaries_temp (SELECT * FROM cclf_beneficiaries WHERE id between 83000000 and 83999999);
INSERT INTO cclf_beneficiaries_temp (SELECT * FROM cclf_beneficiaries WHERE id between 84000000 and 84999999);
INSERT INTO cclf_beneficiaries_temp (SELECT * FROM cclf_beneficiaries WHERE id between 85000000 and 85999999);
INSERT INTO cclf_beneficiaries_temp (SELECT * FROM cclf_beneficiaries WHERE id between 86000000 and 86999999);
INSERT INTO cclf_beneficiaries_temp (SELECT * FROM cclf_beneficiaries WHERE id between 87000000 and 87999999);
INSERT INTO cclf_beneficiaries_temp (SELECT * FROM cclf_beneficiaries WHERE id between 88000000 and 88999999);
INSERT INTO cclf_beneficiaries_temp (SELECT * FROM cclf_beneficiaries WHERE id between 89000000 and 89999999);
INSERT INTO cclf_beneficiaries_temp (SELECT * FROM cclf_beneficiaries WHERE id between 90000000 and 90999999);
INSERT INTO cclf_beneficiaries_temp (SELECT * FROM cclf_beneficiaries WHERE id between 91000000 and 91999999);
INSERT INTO cclf_beneficiaries_temp (SELECT * FROM cclf_beneficiaries WHERE id between 92000000 and 92999999);
INSERT INTO cclf_beneficiaries_temp (SELECT * FROM cclf_beneficiaries WHERE id between 93000000 and 93999999);
INSERT INTO cclf_beneficiaries_temp (SELECT * FROM cclf_beneficiaries WHERE id between 94000000 and 94999999);
INSERT INTO cclf_beneficiaries_temp (SELECT * FROM cclf_beneficiaries WHERE id between 95000000 and 95999999);
INSERT INTO cclf_beneficiaries_temp (SELECT * FROM cclf_beneficiaries WHERE id between 96000000 and 96999999);
INSERT INTO cclf_beneficiaries_temp (SELECT * FROM cclf_beneficiaries WHERE id between 97000000 and 97999999);
INSERT INTO cclf_beneficiaries_temp (SELECT * FROM cclf_beneficiaries WHERE id between 98000000 and 98999999);
INSERT INTO cclf_beneficiaries_temp (SELECT * FROM cclf_beneficiaries WHERE id between 99000000 and 99999999);
INSERT INTO cclf_beneficiaries_temp (SELECT * FROM cclf_beneficiaries WHERE id between 100000000 and 100999999);
INSERT INTO cclf_beneficiaries_temp (SELECT * FROM cclf_beneficiaries WHERE id between 101000000 and 101999999);
INSERT INTO cclf_beneficiaries_temp (SELECT * FROM cclf_beneficiaries WHERE id between 102000000 and 102999999);
INSERT INTO cclf_beneficiaries_temp (SELECT * FROM cclf_beneficiaries WHERE id between 103000000 and 103999999);
INSERT INTO cclf_beneficiaries_temp (SELECT * FROM cclf_beneficiaries WHERE id between 104000000 and 104999999);
INSERT INTO cclf_beneficiaries_temp (SELECT * FROM cclf_beneficiaries WHERE id between 105000000 and 105999999);
INSERT INTO cclf_beneficiaries_temp (SELECT * FROM cclf_beneficiaries WHERE id between 106000000 and 106999999);
INSERT INTO cclf_beneficiaries_temp (SELECT * FROM cclf_beneficiaries WHERE id between 107000000 and 107999999);
INSERT INTO cclf_beneficiaries_temp (SELECT * FROM cclf_beneficiaries WHERE id between 108000000 and 108999999);
INSERT INTO cclf_beneficiaries_temp (SELECT * FROM cclf_beneficiaries WHERE id between 109000000 and 109999999);
INSERT INTO cclf_beneficiaries_temp (SELECT * FROM cclf_beneficiaries WHERE id between 110000000 and 110999999);
INSERT INTO cclf_beneficiaries_temp (SELECT * FROM cclf_beneficiaries WHERE id between 111000000 and 111999999);
INSERT INTO cclf_beneficiaries_temp (SELECT * FROM cclf_beneficiaries WHERE id between 112000000 and 112999999);
INSERT INTO cclf_beneficiaries_temp (SELECT * FROM cclf_beneficiaries WHERE id between 113000000 and 113999999);
INSERT INTO cclf_beneficiaries_temp (SELECT * FROM cclf_beneficiaries WHERE id between 114000000 and 114999999);
INSERT INTO cclf_beneficiaries_temp (SELECT * FROM cclf_beneficiaries WHERE id between 115000000 and 115999999);
INSERT INTO cclf_beneficiaries_temp (SELECT * FROM cclf_beneficiaries WHERE id between 116000000 and 116999999);
INSERT INTO cclf_beneficiaries_temp (SELECT * FROM cclf_beneficiaries WHERE id between 117000000 and 117999999);
INSERT INTO cclf_beneficiaries_temp (SELECT * FROM cclf_beneficiaries WHERE id between 118000000 and 118999999);
INSERT INTO cclf_beneficiaries_temp (SELECT * FROM cclf_beneficiaries WHERE id between 119000000 and 119999999);
INSERT INTO cclf_beneficiaries_temp (SELECT * FROM cclf_beneficiaries WHERE id between 120000000 and 120999999);
INSERT INTO cclf_beneficiaries_temp (SELECT * FROM cclf_beneficiaries WHERE id between 121000000 and 121999999);
INSERT INTO cclf_beneficiaries_temp (SELECT * FROM cclf_beneficiaries WHERE id between 122000000 and 122999999);
INSERT INTO cclf_beneficiaries_temp (SELECT * FROM cclf_beneficiaries WHERE id between 123000000 and 123999999);
INSERT INTO cclf_beneficiaries_temp (SELECT * FROM cclf_beneficiaries WHERE id between 124000000 and 124999999);
INSERT INTO cclf_beneficiaries_temp (SELECT * FROM cclf_beneficiaries WHERE id between 125000000 and 125999999);
INSERT INTO cclf_beneficiaries_temp (SELECT * FROM cclf_beneficiaries WHERE id between 126000000 and 126999999);
INSERT INTO cclf_beneficiaries_temp (SELECT * FROM cclf_beneficiaries WHERE id between 127000000 and 127999999);
INSERT INTO cclf_beneficiaries_temp (SELECT * FROM cclf_beneficiaries WHERE id between 128000000 and 128999999);
INSERT INTO cclf_beneficiaries_temp (SELECT * FROM cclf_beneficiaries WHERE id between 129000000 and 129999999);
INSERT INTO cclf_beneficiaries_temp (SELECT * FROM cclf_beneficiaries WHERE id between 130000000 and 130999999);
INSERT INTO cclf_beneficiaries_temp (SELECT * FROM cclf_beneficiaries WHERE id between 131000000 and 131999999);
INSERT INTO cclf_beneficiaries_temp (SELECT * FROM cclf_beneficiaries WHERE id between 132000000 and 132999999);
INSERT INTO cclf_beneficiaries_temp (SELECT * FROM cclf_beneficiaries WHERE id between 133000000 and 133999999);
INSERT INTO cclf_beneficiaries_temp (SELECT * FROM cclf_beneficiaries WHERE id between 134000000 and 134999999);
INSERT INTO cclf_beneficiaries_temp (SELECT * FROM cclf_beneficiaries WHERE id between 135000000 and 135999999);
INSERT INTO cclf_beneficiaries_temp (SELECT * FROM cclf_beneficiaries WHERE id between 136000000 and 136999999);
INSERT INTO cclf_beneficiaries_temp (SELECT * FROM cclf_beneficiaries WHERE id between 137000000 and 137999999);
INSERT INTO cclf_beneficiaries_temp (SELECT * FROM cclf_beneficiaries WHERE id between 138000000 and 138999999);
INSERT INTO cclf_beneficiaries_temp (SELECT * FROM cclf_beneficiaries WHERE id between 139000000 and 139999999);
INSERT INTO cclf_beneficiaries_temp (SELECT * FROM cclf_beneficiaries WHERE id between 140000000 and 140999999);
INSERT INTO cclf_beneficiaries_temp (SELECT * FROM cclf_beneficiaries WHERE id between 141000000 and 141999999);
INSERT INTO cclf_beneficiaries_temp (SELECT * FROM cclf_beneficiaries WHERE id between 142000000 and 142999999);
INSERT INTO cclf_beneficiaries_temp (SELECT * FROM cclf_beneficiaries WHERE id between 143000000 and 143999999);
INSERT INTO cclf_beneficiaries_temp (SELECT * FROM cclf_beneficiaries WHERE id between 144000000 and 144999999);
INSERT INTO cclf_beneficiaries_temp (SELECT * FROM cclf_beneficiaries WHERE id between 145000000 and 145999999);
INSERT INTO cclf_beneficiaries_temp (SELECT * FROM cclf_beneficiaries WHERE id between 146000000 and 146999999);
INSERT INTO cclf_beneficiaries_temp (SELECT * FROM cclf_beneficiaries WHERE id between 147000000 and 147999999);
INSERT INTO cclf_beneficiaries_temp (SELECT * FROM cclf_beneficiaries WHERE id between 148000000 and 148999999);
INSERT INTO cclf_beneficiaries_temp (SELECT * FROM cclf_beneficiaries WHERE id between 149000000 and 149999999);
INSERT INTO cclf_beneficiaries_temp (SELECT * FROM cclf_beneficiaries WHERE id between 150000000 and 150999999);
INSERT INTO cclf_beneficiaries_temp (SELECT * FROM cclf_beneficiaries WHERE id between 151000000 and 151999999);
INSERT INTO cclf_beneficiaries_temp (SELECT * FROM cclf_beneficiaries WHERE id between 152000000 and 152999999);
INSERT INTO cclf_beneficiaries_temp (SELECT * FROM cclf_beneficiaries WHERE id between 153000000 and 153999999);
INSERT INTO cclf_beneficiaries_temp (SELECT * FROM cclf_beneficiaries WHERE id between 154000000 and 154999999);
INSERT INTO cclf_beneficiaries_temp (SELECT * FROM cclf_beneficiaries WHERE id between 155000000 and 155999999);
INSERT INTO cclf_beneficiaries_temp (SELECT * FROM cclf_beneficiaries WHERE id between 156000000 and 156999999);
INSERT INTO cclf_beneficiaries_temp (SELECT * FROM cclf_beneficiaries WHERE id between 157000000 and 157999999);
INSERT INTO cclf_beneficiaries_temp (SELECT * FROM cclf_beneficiaries WHERE id between 158000000 and 158999999);
INSERT INTO cclf_beneficiaries_temp (SELECT * FROM cclf_beneficiaries WHERE id between 159000000 and 159999999);
INSERT INTO cclf_beneficiaries_temp (SELECT * FROM cclf_beneficiaries WHERE id between 160000000 and 160999999);
INSERT INTO cclf_beneficiaries_temp (SELECT * FROM cclf_beneficiaries WHERE id between 161000000 and 161999999);
INSERT INTO cclf_beneficiaries_temp (SELECT * FROM cclf_beneficiaries WHERE id between 162000000 and 162999999);
INSERT INTO cclf_beneficiaries_temp (SELECT * FROM cclf_beneficiaries WHERE id between 163000000 and 163999999);
INSERT INTO cclf_beneficiaries_temp (SELECT * FROM cclf_beneficiaries WHERE id between 164000000 and 164999999);
INSERT INTO cclf_beneficiaries_temp (SELECT * FROM cclf_beneficiaries WHERE id between 165000000 and 165999999);
INSERT INTO cclf_beneficiaries_temp (SELECT * FROM cclf_beneficiaries WHERE id between 166000000 and 166999999);
INSERT INTO cclf_beneficiaries_temp (SELECT * FROM cclf_beneficiaries WHERE id between 167000000 and 167999999);
INSERT INTO cclf_beneficiaries_temp (SELECT * FROM cclf_beneficiaries WHERE id between 168000000 and 168999999);
INSERT INTO cclf_beneficiaries_temp (SELECT * FROM cclf_beneficiaries WHERE id between 169000000 and 169999999);
INSERT INTO cclf_beneficiaries_temp (SELECT * FROM cclf_beneficiaries WHERE id between 170000000 and 170999999);
INSERT INTO cclf_beneficiaries_temp (SELECT * FROM cclf_beneficiaries WHERE id between 171000000 and 171999999);
INSERT INTO cclf_beneficiaries_temp (SELECT * FROM cclf_beneficiaries WHERE id between 172000000 and 172999999);
INSERT INTO cclf_beneficiaries_temp (SELECT * FROM cclf_beneficiaries WHERE id between 173000000 and 173999999);
INSERT INTO cclf_beneficiaries_temp (SELECT * FROM cclf_beneficiaries WHERE id between 174000000 and 174999999);
INSERT INTO cclf_beneficiaries_temp (SELECT * FROM cclf_beneficiaries WHERE id between 175000000 and 175999999);
INSERT INTO cclf_beneficiaries_temp (SELECT * FROM cclf_beneficiaries WHERE id between 176000000 and 176999999);
INSERT INTO cclf_beneficiaries_temp (SELECT * FROM cclf_beneficiaries WHERE id between 177000000 and 177999999);
INSERT INTO cclf_beneficiaries_temp (SELECT * FROM cclf_beneficiaries WHERE id between 178000000 and 178999999);
INSERT INTO cclf_beneficiaries_temp (SELECT * FROM cclf_beneficiaries WHERE id between 179000000 and 179999999);
INSERT INTO cclf_beneficiaries_temp (SELECT * FROM cclf_beneficiaries WHERE id between 180000000 and 180999999);
INSERT INTO cclf_beneficiaries_temp (SELECT * FROM cclf_beneficiaries WHERE id between 181000000 and 181999999);
INSERT INTO cclf_beneficiaries_temp (SELECT * FROM cclf_beneficiaries WHERE id between 182000000 and 182999999);
INSERT INTO cclf_beneficiaries_temp (SELECT * FROM cclf_beneficiaries WHERE id between 183000000 and 183999999);
INSERT INTO cclf_beneficiaries_temp (SELECT * FROM cclf_beneficiaries WHERE id between 184000000 and 184999999);
INSERT INTO cclf_beneficiaries_temp (SELECT * FROM cclf_beneficiaries WHERE id between 185000000 and 185999999);
INSERT INTO cclf_beneficiaries_temp (SELECT * FROM cclf_beneficiaries WHERE id between 186000000 and 186999999);
INSERT INTO cclf_beneficiaries_temp (SELECT * FROM cclf_beneficiaries WHERE id between 187000000 and 187999999);
INSERT INTO cclf_beneficiaries_temp (SELECT * FROM cclf_beneficiaries WHERE id between 188000000 and 188999999);
INSERT INTO cclf_beneficiaries_temp (SELECT * FROM cclf_beneficiaries WHERE id between 189000000 and 189999999);
INSERT INTO cclf_beneficiaries_temp (SELECT * FROM cclf_beneficiaries WHERE id between 190000000 and 190999999);
INSERT INTO cclf_beneficiaries_temp (SELECT * FROM cclf_beneficiaries WHERE id between 191000000 and 191999999);
INSERT INTO cclf_beneficiaries_temp (SELECT * FROM cclf_beneficiaries WHERE id between 192000000 and 192999999);
INSERT INTO cclf_beneficiaries_temp (SELECT * FROM cclf_beneficiaries WHERE id between 193000000 and 193999999);
INSERT INTO cclf_beneficiaries_temp (SELECT * FROM cclf_beneficiaries WHERE id between 194000000 and 194999999);
INSERT INTO cclf_beneficiaries_temp (SELECT * FROM cclf_beneficiaries WHERE id between 195000000 and 195999999);
INSERT INTO cclf_beneficiaries_temp (SELECT * FROM cclf_beneficiaries WHERE id between 196000000 and 196999999);
INSERT INTO cclf_beneficiaries_temp (SELECT * FROM cclf_beneficiaries WHERE id between 197000000 and 197999999);
INSERT INTO cclf_beneficiaries_temp (SELECT * FROM cclf_beneficiaries WHERE id between 198000000 and 198999999);
INSERT INTO cclf_beneficiaries_temp (SELECT * FROM cclf_beneficiaries WHERE id between 199000000 and 199999999);
INSERT INTO cclf_beneficiaries_temp (SELECT * FROM cclf_beneficiaries WHERE id between 200000000 and 200999999);
INSERT INTO cclf_beneficiaries_temp (SELECT * FROM cclf_beneficiaries WHERE id between 201000000 and 201999999);
INSERT INTO cclf_beneficiaries_temp (SELECT * FROM cclf_beneficiaries WHERE id between 202000000 and 202999999);
INSERT INTO cclf_beneficiaries_temp (SELECT * FROM cclf_beneficiaries WHERE id between 203000000 and 203999999);
INSERT INTO cclf_beneficiaries_temp (SELECT * FROM cclf_beneficiaries WHERE id between 204000000 and 204999999);
INSERT INTO cclf_beneficiaries_temp (SELECT * FROM cclf_beneficiaries WHERE id between 205000000 and 205999999);
INSERT INTO cclf_beneficiaries_temp (SELECT * FROM cclf_beneficiaries WHERE id between 206000000 and 206999999);
INSERT INTO cclf_beneficiaries_temp (SELECT * FROM cclf_beneficiaries WHERE id between 207000000 and 207999999);
INSERT INTO cclf_beneficiaries_temp (SELECT * FROM cclf_beneficiaries WHERE id between 208000000 and 208999999);
INSERT INTO cclf_beneficiaries_temp (SELECT * FROM cclf_beneficiaries WHERE id between 209000000 and 209999999);
INSERT INTO cclf_beneficiaries_temp (SELECT * FROM cclf_beneficiaries WHERE id between 210000000 and 210999999);
INSERT INTO cclf_beneficiaries_temp (SELECT * FROM cclf_beneficiaries WHERE id between 211000000 and 211999999);
INSERT INTO cclf_beneficiaries_temp (SELECT * FROM cclf_beneficiaries WHERE id between 212000000 and 212999999);
INSERT INTO cclf_beneficiaries_temp (SELECT * FROM cclf_beneficiaries WHERE id between 213000000 and 213999999);
INSERT INTO cclf_beneficiaries_temp (SELECT * FROM cclf_beneficiaries WHERE id between 214000000 and 214999999);
INSERT INTO cclf_beneficiaries_temp (SELECT * FROM cclf_beneficiaries WHERE id between 215000000 and 215999999);
INSERT INTO cclf_beneficiaries_temp (SELECT * FROM cclf_beneficiaries WHERE id between 216000000 and 216999999);
INSERT INTO cclf_beneficiaries_temp (SELECT * FROM cclf_beneficiaries WHERE id between 217000000 and 217999999);
INSERT INTO cclf_beneficiaries_temp (SELECT * FROM cclf_beneficiaries WHERE id between 218000000 and 218999999);
INSERT INTO cclf_beneficiaries_temp (SELECT * FROM cclf_beneficiaries WHERE id between 219000000 and 219999999);
INSERT INTO cclf_beneficiaries_temp (SELECT * FROM cclf_beneficiaries WHERE id between 220000000 and 220999999);
INSERT INTO cclf_beneficiaries_temp (SELECT * FROM cclf_beneficiaries WHERE id between 221000000 and 221999999);
INSERT INTO cclf_beneficiaries_temp (SELECT * FROM cclf_beneficiaries WHERE id between 222000000 and 222999999);
INSERT INTO cclf_beneficiaries_temp (SELECT * FROM cclf_beneficiaries WHERE id between 223000000 and 223999999);
INSERT INTO cclf_beneficiaries_temp (SELECT * FROM cclf_beneficiaries WHERE id between 224000000 and 224999999);
INSERT INTO cclf_beneficiaries_temp (SELECT * FROM cclf_beneficiaries WHERE id between 225000000 and 225999999);
INSERT INTO cclf_beneficiaries_temp (SELECT * FROM cclf_beneficiaries WHERE id between 226000000 and 226999999);
INSERT INTO cclf_beneficiaries_temp (SELECT * FROM cclf_beneficiaries WHERE id between 227000000 and 227999999);
INSERT INTO cclf_beneficiaries_temp (SELECT * FROM cclf_beneficiaries WHERE id between 228000000 and 228999999);
INSERT INTO cclf_beneficiaries_temp (SELECT * FROM cclf_beneficiaries WHERE id between 229000000 and 229999999);
INSERT INTO cclf_beneficiaries_temp (SELECT * FROM cclf_beneficiaries WHERE id between 230000000 and 230999999);
INSERT INTO cclf_beneficiaries_temp (SELECT * FROM cclf_beneficiaries WHERE id between 231000000 and 231999999);
INSERT INTO cclf_beneficiaries_temp (SELECT * FROM cclf_beneficiaries WHERE id between 232000000 and 232999999);
INSERT INTO cclf_beneficiaries_temp (SELECT * FROM cclf_beneficiaries WHERE id between 233000000 and 233999999);
INSERT INTO cclf_beneficiaries_temp (SELECT * FROM cclf_beneficiaries WHERE id between 234000000 and 234999999);
INSERT INTO cclf_beneficiaries_temp (SELECT * FROM cclf_beneficiaries WHERE id between 235000000 and 235999999);
INSERT INTO cclf_beneficiaries_temp (SELECT * FROM cclf_beneficiaries WHERE id between 236000000 and 236999999);
INSERT INTO cclf_beneficiaries_temp (SELECT * FROM cclf_beneficiaries WHERE id between 237000000 and 237999999);
INSERT INTO cclf_beneficiaries_temp (SELECT * FROM cclf_beneficiaries WHERE id between 238000000 and 238999999);
INSERT INTO cclf_beneficiaries_temp (SELECT * FROM cclf_beneficiaries WHERE id between 239000000 and 239999999);
INSERT INTO cclf_beneficiaries_temp (SELECT * FROM cclf_beneficiaries WHERE id between 240000000 and 240999999);
INSERT INTO cclf_beneficiaries_temp (SELECT * FROM cclf_beneficiaries WHERE id between 241000000 and 241999999);
INSERT INTO cclf_beneficiaries_temp (SELECT * FROM cclf_beneficiaries WHERE id between 242000000 and 242999999);
INSERT INTO cclf_beneficiaries_temp (SELECT * FROM cclf_beneficiaries WHERE id between 243000000 and 243999999);
INSERT INTO cclf_beneficiaries_temp (SELECT * FROM cclf_beneficiaries WHERE id between 244000000 and 244999999);
INSERT INTO cclf_beneficiaries_temp (SELECT * FROM cclf_beneficiaries WHERE id between 245000000 and 245999999);
INSERT INTO cclf_beneficiaries_temp (SELECT * FROM cclf_beneficiaries WHERE id between 246000000 and 246999999);
INSERT INTO cclf_beneficiaries_temp (SELECT * FROM cclf_beneficiaries WHERE id between 247000000 and 247999999);
INSERT INTO cclf_beneficiaries_temp (SELECT * FROM cclf_beneficiaries WHERE id between 248000000 and 248999999);
INSERT INTO cclf_beneficiaries_temp (SELECT * FROM cclf_beneficiaries WHERE id between 249000000 and 249999999);
INSERT INTO cclf_beneficiaries_temp (SELECT * FROM cclf_beneficiaries WHERE id between 250000000 and 250999999);
INSERT INTO cclf_beneficiaries_temp (SELECT * FROM cclf_beneficiaries WHERE id between 251000000 and 251999999);
INSERT INTO cclf_beneficiaries_temp (SELECT * FROM cclf_beneficiaries WHERE id between 252000000 and 252999999);
INSERT INTO cclf_beneficiaries_temp (SELECT * FROM cclf_beneficiaries WHERE id between 253000000 and 253999999);
INSERT INTO cclf_beneficiaries_temp (SELECT * FROM cclf_beneficiaries WHERE id between 254000000 and 254999999);
INSERT INTO cclf_beneficiaries_temp (SELECT * FROM cclf_beneficiaries WHERE id between 255000000 and 255999999);
INSERT INTO cclf_beneficiaries_temp (SELECT * FROM cclf_beneficiaries WHERE id between 256000000 and 256999999);
INSERT INTO cclf_beneficiaries_temp (SELECT * FROM cclf_beneficiaries WHERE id between 257000000 and 257999999);
INSERT INTO cclf_beneficiaries_temp (SELECT * FROM cclf_beneficiaries WHERE id between 258000000 and 258999999);
INSERT INTO cclf_beneficiaries_temp (SELECT * FROM cclf_beneficiaries WHERE id between 259000000 and 259999999);
INSERT INTO cclf_beneficiaries_temp (SELECT * FROM cclf_beneficiaries WHERE id between 260000000 and 260999999);
INSERT INTO cclf_beneficiaries_temp (SELECT * FROM cclf_beneficiaries WHERE id between 261000000 and 261999999);
INSERT INTO cclf_beneficiaries_temp (SELECT * FROM cclf_beneficiaries WHERE id between 262000000 and 262999999);
INSERT INTO cclf_beneficiaries_temp (SELECT * FROM cclf_beneficiaries WHERE id between 263000000 and 263999999);
INSERT INTO cclf_beneficiaries_temp (SELECT * FROM cclf_beneficiaries WHERE id between 264000000 and 264999999);
INSERT INTO cclf_beneficiaries_temp (SELECT * FROM cclf_beneficiaries WHERE id between 265000000 and 265999999);
INSERT INTO cclf_beneficiaries_temp (SELECT * FROM cclf_beneficiaries WHERE id between 266000000 and 266999999);
INSERT INTO cclf_beneficiaries_temp (SELECT * FROM cclf_beneficiaries WHERE id between 267000000 and 267999999);
INSERT INTO cclf_beneficiaries_temp (SELECT * FROM cclf_beneficiaries WHERE id between 268000000 and 268999999);
INSERT INTO cclf_beneficiaries_temp (SELECT * FROM cclf_beneficiaries WHERE id between 269000000 and 269999999);
INSERT INTO cclf_beneficiaries_temp (SELECT * FROM cclf_beneficiaries WHERE id between 270000000 and 270999999);
INSERT INTO cclf_beneficiaries_temp (SELECT * FROM cclf_beneficiaries WHERE id between 271000000 and 271999999);
INSERT INTO cclf_beneficiaries_temp (SELECT * FROM cclf_beneficiaries WHERE id between 272000000 and 272999999);
INSERT INTO cclf_beneficiaries_temp (SELECT * FROM cclf_beneficiaries WHERE id between 273000000 and 273999999);
INSERT INTO cclf_beneficiaries_temp (SELECT * FROM cclf_beneficiaries WHERE id between 274000000 and 274999999);
INSERT INTO cclf_beneficiaries_temp (SELECT * FROM cclf_beneficiaries WHERE id between 275000000 and 275999999);
INSERT INTO cclf_beneficiaries_temp (SELECT * FROM cclf_beneficiaries WHERE id between 276000000 and 276999999);
INSERT INTO cclf_beneficiaries_temp (SELECT * FROM cclf_beneficiaries WHERE id between 277000000 and 277999999);
INSERT INTO cclf_beneficiaries_temp (SELECT * FROM cclf_beneficiaries WHERE id between 278000000 and 278999999);
INSERT INTO cclf_beneficiaries_temp (SELECT * FROM cclf_beneficiaries WHERE id between 279000000 and 279999999);
INSERT INTO cclf_beneficiaries_temp (SELECT * FROM cclf_beneficiaries WHERE id between 280000000 and 280999999);
INSERT INTO cclf_beneficiaries_temp (SELECT * FROM cclf_beneficiaries WHERE id between 281000000 and 281999999);
INSERT INTO cclf_beneficiaries_temp (SELECT * FROM cclf_beneficiaries WHERE id between 282000000 and 282999999);
INSERT INTO cclf_beneficiaries_temp (SELECT * FROM cclf_beneficiaries WHERE id between 283000000 and 283999999);
INSERT INTO cclf_beneficiaries_temp (SELECT * FROM cclf_beneficiaries WHERE id between 284000000 and 284999999);
INSERT INTO cclf_beneficiaries_temp (SELECT * FROM cclf_beneficiaries WHERE id between 285000000 and 285999999);
INSERT INTO cclf_beneficiaries_temp (SELECT * FROM cclf_beneficiaries WHERE id between 286000000 and 286999999);
INSERT INTO cclf_beneficiaries_temp (SELECT * FROM cclf_beneficiaries WHERE id between 287000000 and 287999999);
INSERT INTO cclf_beneficiaries_temp (SELECT * FROM cclf_beneficiaries WHERE id between 288000000 and 288999999);
INSERT INTO cclf_beneficiaries_temp (SELECT * FROM cclf_beneficiaries WHERE id between 289000000 and 289999999);
INSERT INTO cclf_beneficiaries_temp (SELECT * FROM cclf_beneficiaries WHERE id between 290000000 and 290999999);
INSERT INTO cclf_beneficiaries_temp (SELECT * FROM cclf_beneficiaries WHERE id between 291000000 and 291999999);
INSERT INTO cclf_beneficiaries_temp (SELECT * FROM cclf_beneficiaries WHERE id between 292000000 and 292999999);
INSERT INTO cclf_beneficiaries_temp (SELECT * FROM cclf_beneficiaries WHERE id between 293000000 and 293999999);
INSERT INTO cclf_beneficiaries_temp (SELECT * FROM cclf_beneficiaries WHERE id between 294000000 and 294999999);
INSERT INTO cclf_beneficiaries_temp (SELECT * FROM cclf_beneficiaries WHERE id between 295000000 and 295999999);
INSERT INTO cclf_beneficiaries_temp (SELECT * FROM cclf_beneficiaries WHERE id between 296000000 and 296999999);
INSERT INTO cclf_beneficiaries_temp (SELECT * FROM cclf_beneficiaries WHERE id between 297000000 and 297999999);
INSERT INTO cclf_beneficiaries_temp (SELECT * FROM cclf_beneficiaries WHERE id between 298000000 and 298999999);
INSERT INTO cclf_beneficiaries_temp (SELECT * FROM cclf_beneficiaries WHERE id between 299000000 and 299999999);
INSERT INTO cclf_beneficiaries_temp (SELECT * FROM cclf_beneficiaries WHERE id between 300000000 and 300999999);
INSERT INTO cclf_beneficiaries_temp (SELECT * FROM cclf_beneficiaries WHERE id between 301000000 and 301999999);
INSERT INTO cclf_beneficiaries_temp (SELECT * FROM cclf_beneficiaries WHERE id between 302000000 and 302999999);
INSERT INTO cclf_beneficiaries_temp (SELECT * FROM cclf_beneficiaries WHERE id between 303000000 and 303999999);
INSERT INTO cclf_beneficiaries_temp (SELECT * FROM cclf_beneficiaries WHERE id between 304000000 and 304999999);
INSERT INTO cclf_beneficiaries_temp (SELECT * FROM cclf_beneficiaries WHERE id between 305000000 and 305999999);
INSERT INTO cclf_beneficiaries_temp (SELECT * FROM cclf_beneficiaries WHERE id between 306000000 and 306999999);
INSERT INTO cclf_beneficiaries_temp (SELECT * FROM cclf_beneficiaries WHERE id between 307000000 and 307999999);
INSERT INTO cclf_beneficiaries_temp (SELECT * FROM cclf_beneficiaries WHERE id between 308000000 and 308999999);
INSERT INTO cclf_beneficiaries_temp (SELECT * FROM cclf_beneficiaries WHERE id between 309000000 and 309999999);
INSERT INTO cclf_beneficiaries_temp (SELECT * FROM cclf_beneficiaries WHERE id between 310000000 and 310999999);
INSERT INTO cclf_beneficiaries_temp (SELECT * FROM cclf_beneficiaries WHERE id between 311000000 and 311999999);
INSERT INTO cclf_beneficiaries_temp (SELECT * FROM cclf_beneficiaries WHERE id between 312000000 and 312999999);
INSERT INTO cclf_beneficiaries_temp (SELECT * FROM cclf_beneficiaries WHERE id between 313000000 and 313999999);
INSERT INTO cclf_beneficiaries_temp (SELECT * FROM cclf_beneficiaries WHERE id between 314000000 and 314999999);
INSERT INTO cclf_beneficiaries_temp (SELECT * FROM cclf_beneficiaries WHERE id between 315000000 and 315999999);
INSERT INTO cclf_beneficiaries_temp (SELECT * FROM cclf_beneficiaries WHERE id between 316000000 and 316999999);
INSERT INTO cclf_beneficiaries_temp (SELECT * FROM cclf_beneficiaries WHERE id between 317000000 and 317999999);
INSERT INTO cclf_beneficiaries_temp (SELECT * FROM cclf_beneficiaries WHERE id between 318000000 and 318999999);
INSERT INTO cclf_beneficiaries_temp (SELECT * FROM cclf_beneficiaries WHERE id between 319000000 and 319999999);
INSERT INTO cclf_beneficiaries_temp (SELECT * FROM cclf_beneficiaries WHERE id between 320000000 and 320999999);
INSERT INTO cclf_beneficiaries_temp (SELECT * FROM cclf_beneficiaries WHERE id between 321000000 and 321999999);
INSERT INTO cclf_beneficiaries_temp (SELECT * FROM cclf_beneficiaries WHERE id between 322000000 and 322999999);
INSERT INTO cclf_beneficiaries_temp (SELECT * FROM cclf_beneficiaries WHERE id between 323000000 and 323999999);
INSERT INTO cclf_beneficiaries_temp (SELECT * FROM cclf_beneficiaries WHERE id between 324000000 and 324999999);
INSERT INTO cclf_beneficiaries_temp (SELECT * FROM cclf_beneficiaries WHERE id between 325000000 and 325999999);
INSERT INTO cclf_beneficiaries_temp (SELECT * FROM cclf_beneficiaries WHERE id between 326000000 and 326999999);
INSERT INTO cclf_beneficiaries_temp (SELECT * FROM cclf_beneficiaries WHERE id between 327000000 and 327999999);
INSERT INTO cclf_beneficiaries_temp (SELECT * FROM cclf_beneficiaries WHERE id between 328000000 and 329244316);

ALTER TABLE ONLY public.cclf_beneficiaries_temp ADD CONSTRAINT cclf_beneficiaries_temp_pkey PRIMARY KEY (id);

CREATE INDEX IF NOT EXISTS idx_cclf_beneficiaries_temp_file_id ON public.cclf_beneficiaries_temp USING btree (file_id);

ALTER TABLE ONLY public.cclf_beneficiaries_temp
    ADD CONSTRAINT cclf_beneficiaries_temp_file_id_cclf_files_id_foreign FOREIGN KEY (file_id) REFERENCES public.cclf_files(id) ON UPDATE RESTRICT ON DELETE RESTRICT;

ALTER TABLE public.cclf_beneficiaries_temp ALTER COLUMN created_at SET DEFAULT now();
ALTER TABLE public.cclf_beneficiaries_temp ALTER COLUMN updated_at SET DEFAULT now();
ALTER TABLE ONLY public.cclf_beneficiaries_temp ALTER COLUMN id SET DEFAULT nextval('public.cclf_beneficiaries_id_seq'::regclass);

BEGIN;
ALTER TABLE cclf_beneficiaries RENAME TO cclf_beneficiaries_old;
ALTER TABLE cclf_beneficiaries_temp RENAME TO cclf_beneficiaries;
COMMIT;

ALTER SEQUENCE cclf_beneficiaries_id_seq OWNED BY cclf_beneficiaries.id;

BEGIN;
CREATE TRIGGER set_timestamp
BEFORE UPDATE ON public.cclf_beneficiaries
FOR EACH ROW
EXECUTE PROCEDURE trigger_set_timestamp();
COMMIT;

DROP TABLE cclf_beneficiaries_old;

ALTER INDEX cclf_beneficiaries_temp_pkey RENAME TO cclf_beneficiaries_pkey;
ALTER INDEX idx_cclf_beneficiaries_temp_file_id RENAME TO idx_cclf_beneficiaries_file_id;
ALTER TABLE cclf_beneficiaries RENAME CONSTRAINT cclf_beneficiaries_temp_file_id_cclf_files_id_foreign TO cclf_beneficiaries_file_id_cclf_files_id_foreign;