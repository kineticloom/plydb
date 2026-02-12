-- 1. Create the Master Nodes (The 'Why' behind the data)
CREATE TABLE vortex_anchor (
    anchor_id UUID PRIMARY KEY,
    designation VARCHAR(50) NOT NULL,
    stability_threshold DECIMAL(10, 4)
);

COMMENT ON TABLE vortex_anchor IS 'Primary stability points for aetheric harvesting. Anchors must remain above their stability_threshold to prevent collapse.';
COMMENT ON COLUMN vortex_anchor.designation IS 'The unique resonant name of the anchor.';

-- 2. The Time Series Table (High-frequency telemetry)
CREATE TABLE flux_telemetry (
    telemetry_id SERIAL PRIMARY KEY,
    anchor_ref UUID REFERENCES vortex_anchor(anchor_id),
    recorded_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    oscill_rate FLOAT8,
    entropy_delta FLOAT8
);

COMMENT ON TABLE flux_telemetry IS 'Time-series log of multidimensional energy fluctuations.';
COMMENT ON COLUMN flux_telemetry.oscill_rate IS 'The frequency of energy vibration. Optimal range is between 400 and 600 mHz.';
COMMENT ON COLUMN flux_telemetry.entropy_delta IS 'The rate of energy decay. Positive values indicate system leakage.';

-- 3. The Obfuscated Relationship Table (Bridge Table)
CREATE TABLE syn_link_01 (
    link_id SERIAL PRIMARY KEY,
    alpha_node UUID REFERENCES vortex_anchor(anchor_id),
    beta_node UUID REFERENCES vortex_anchor(anchor_id),
    conductivity_ratio DECIMAL(5, 2)
);

COMMENT ON TABLE syn_link_01 IS 'Maps the entanglement between two different vortex anchors. High conductivity allows for energy sharing.';

-- ---------------------------------------------------------
-- SEED DATA
-- ---------------------------------------------------------

INSERT INTO vortex_anchor (anchor_id, designation, stability_threshold) VALUES
('550e8400-e29b-41d4-a716-446655440000', 'Obsidian-Nine', 0.8521),
('671e8400-e29b-41d4-a716-446655440001', 'Aether-Prime', 0.1205),
('782e8400-e29b-41d4-a716-446655440002', 'Void-Echo', 0.9999);

INSERT INTO flux_telemetry (anchor_ref, recorded_at, oscill_rate, entropy_delta) VALUES
('550e8400-e29b-41d4-a716-446655440000', NOW() - INTERVAL '10 minutes', 450.5, 0.002),
('550e8400-e29b-41d4-a716-446655440000', NOW() - INTERVAL '5 minutes', 455.2, 0.003),
('550e8400-e29b-41d4-a716-446655440000', NOW(), 448.1, 0.005),
('671e8400-e29b-41d4-a716-446655440001', NOW() - INTERVAL '2 minutes', 590.0, -0.001),
('782e8400-e29b-41d4-a716-446655440002', NOW(), 310.4, 0.150); -- Critical failure state

INSERT INTO syn_link_01 (alpha_node, beta_node, conductivity_ratio) VALUES
('550e8400-e29b-41d4-a716-446655440000', '671e8400-e29b-41d4-a716-446655440001', 0.94),
('671e8400-e29b-41d4-a716-446655440001', '782e8400-e29b-41d4-a716-446655440002', 0.22);
