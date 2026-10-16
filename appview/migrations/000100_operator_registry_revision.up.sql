ALTER TABLE safety_operators ADD COLUMN revision BIGINT NOT NULL DEFAULT 0 CHECK (revision >= 0);
CREATE TRIGGER safety_operators_revision BEFORE UPDATE ON safety_operators
FOR EACH ROW EXECUTE FUNCTION increment_operator_resource_revision();
