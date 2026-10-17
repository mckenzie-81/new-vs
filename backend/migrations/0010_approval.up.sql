ALTER TABLE elections ADD COLUMN approved_by UUID REFERENCES users(id);
ALTER TABLE elections ADD COLUMN rejection_reason TEXT;
