CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- Companies table (tenants)
CREATE TABLE companies (
    company_id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name VARCHAR(255) NOT NULL,
    logo_url TEXT,
    config JSONB DEFAULT '{}',
    default_start_time TIME DEFAULT '10:00:00',
    default_end_time TIME,
    work_hours_per_week NUMERIC(4,1) DEFAULT 42.0,
    lunch_start TIME DEFAULT '13:00:00',
    lunch_end TIME DEFAULT '14:00:00',
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    deleted_at TIMESTAMP WITH TIME ZONE
);

-- Branches table (sucursales/talleres)
CREATE TABLE branches (
    branch_id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    company_id UUID NOT NULL REFERENCES companies(company_id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    address TEXT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    deleted_at TIMESTAMP WITH TIME ZONE
);

-- Users table (trabajadores)
CREATE TABLE users (
    user_id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    company_id UUID NOT NULL REFERENCES companies(company_id) ON DELETE CASCADE,
    branch_id UUID REFERENCES branches(branch_id) ON DELETE SET NULL,
    name VARCHAR(255) NOT NULL,
    email VARCHAR(255) NOT NULL UNIQUE,
    password TEXT NOT NULL,
    rut VARCHAR(20),
    card_uid TEXT,
    rol VARCHAR(50) NOT NULL DEFAULT 'worker' CHECK (rol IN ('admin', 'manager', 'worker', 'super_admin')),
    worker_type VARCHAR(20) DEFAULT 'fixed' CHECK (worker_type IN ('fixed', 'flexible', 'external')),
    expected_hours_per_day DECIMAL(4,2) DEFAULT 8.0,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    deleted_at TIMESTAMP WITH TIME ZONE
);

-- Admin users table (super-admin del SaaS)
CREATE TABLE admin_users (
    admin_id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    email VARCHAR(255) NOT NULL UNIQUE,
    password TEXT NOT NULL,
    name VARCHAR(255) NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Work shifts table (turnos)
CREATE TABLE work_shifts (
    shift_id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    company_id UUID NOT NULL REFERENCES companies(company_id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    days TEXT[] NOT NULL,
    start_time TIME NOT NULL,
    end_time TIME NOT NULL,
    shift_type VARCHAR(20) DEFAULT 'fixed' CHECK (shift_type IN ('fixed', 'rotating', 'flexible')),
    pattern_id UUID,
    lunch_start TIME,
    lunch_end TIME,
    is_active BOOLEAN DEFAULT true,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    deleted_at TIMESTAMP WITH TIME ZONE
);

-- User shifts assignment
CREATE TABLE user_shifts (
    user_shift_id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL REFERENCES users(user_id) ON DELETE CASCADE,
    shift_id UUID NOT NULL REFERENCES work_shifts(shift_id) ON DELETE CASCADE,
    start_date DATE,
    end_date DATE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Shift patterns table (turnos rotativos: 4x3, 3x2, etc)
CREATE TABLE shift_patterns (
    pattern_id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    company_id UUID NOT NULL REFERENCES companies(company_id) ON DELETE CASCADE,
    name VARCHAR(50) NOT NULL,
    work_days INT NOT NULL CHECK (work_days >= 1),
    off_days INT NOT NULL CHECK (off_days >= 0),
    is_legal_modality BOOLEAN DEFAULT false,
    legal_reference VARCHAR(255),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE INDEX idx_shift_patterns_company ON shift_patterns(company_id);

-- Add FK constraint to work_shifts.pattern_id
ALTER TABLE work_shifts ADD CONSTRAINT fk_work_shifts_pattern
    FOREIGN KEY (pattern_id) REFERENCES shift_patterns(pattern_id) ON DELETE SET NULL;

-- Attendance logs table
CREATE TABLE attendance_logs (
    log_id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL REFERENCES users(user_id) ON DELETE CASCADE,
    branch_id UUID NOT NULL REFERENCES branches(branch_id) ON DELETE CASCADE,
    type VARCHAR(20) NOT NULL CHECK (type IN ('checkin', 'checkout')),
    timestamp TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    is_late INT DEFAULT 0, -- minutes late (0 if early or checkout, or for flexible/external workers)
    source VARCHAR(50) DEFAULT 'mobile',
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE INDEX idx_attendance_logs_user_id ON attendance_logs(user_id);
CREATE INDEX idx_attendance_logs_branch_id ON attendance_logs(branch_id);
CREATE INDEX idx_attendance_logs_timestamp ON attendance_logs(timestamp);
CREATE INDEX idx_attendance_logs_user_timestamp ON attendance_logs(user_id, timestamp);

-- Weekly hours summary (pre-calculated)
CREATE TABLE weekly_hours_summary (
    summary_id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL REFERENCES users(user_id) ON DELETE CASCADE,
    week_start DATE NOT NULL,
    total_hours DECIMAL(6,2) DEFAULT 0,
    expected_hours DECIMAL(6,2) DEFAULT 40.0,
    calculated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    UNIQUE(user_id, week_start)
);

-- Monthly arrears summary (pre-calculated)
CREATE TABLE monthly_arrears_summary (
    summary_id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL REFERENCES users(user_id) ON DELETE CASCADE,
    year INT NOT NULL,
    month INT NOT NULL CHECK (month >= 1 AND month <= 12),
    total_arrears_minutes INT DEFAULT 0,
    days_with_arrears INT DEFAULT 0,
    calculated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    UNIQUE(user_id, year, month)
);

-- Refresh tokens table
CREATE TABLE refresh_tokens (
    token_id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL REFERENCES users(user_id) ON DELETE CASCADE,
    token TEXT NOT NULL UNIQUE,
    expires_at TIMESTAMP WITH TIME ZONE NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE INDEX idx_refresh_tokens_user_id ON refresh_tokens(user_id);
CREATE INDEX idx_refresh_tokens_token ON refresh_tokens(token);

-- Function to calculate weekly hours for a user
CREATE OR REPLACE FUNCTION calculate_weekly_hours(p_user_id UUID, p_week_start DATE)
RETURNS NUMERIC AS $$
DECLARE
    v_total_minutes INTEGER := 0;
    v_checkin_time TIMESTAMP;
    v_checkout_time TIMESTAMP;
    v_week_end DATE;
    v_r RECORD;
BEGIN
    v_week_end := p_week_start + INTERVAL '6 days';

    FOR v_r IN
        SELECT
            al_checkin.timestamp as checkin_ts,
            al_checkout.timestamp as checkout_ts
        FROM attendance_logs al_checkin
        INNER JOIN attendance_logs al_checkout ON
            al_checkin.user_id = al_checkout.user_id AND
            al_checkin.type = 'checkin' AND
            al_checkout.type = 'checkout' AND
            al_checkin.timestamp < al_checkout.timestamp AND
            al_checkin.timestamp >= p_week_start AND
            al_checkin.timestamp <= v_week_end
        WHERE al_checkin.user_id = p_user_id
        ORDER BY al_checkin.timestamp
    LOOP
        v_checkin_time := v_r.checkin_ts;
        v_checkout_time := v_r.checkout_ts;
        v_total_minutes := v_total_minutes + EXTRACT(EPOCH FROM (v_checkout_time - v_checkin_time))/60;
    END LOOP;

    RETURN ROUND(v_total_minutes / 60.0, 2);
END;
$$ LANGUAGE plpgsql;

-- Function to calculate monthly arrears: sum of minutes late only (not worked deficit)
-- Lateness = checkin time - default_start time (only if checkin was after default_start)
CREATE OR REPLACE FUNCTION calculate_monthly_arrears(p_user_id UUID, p_year INT, p_month INT)
RETURNS TABLE(total_arrears_minutes INT, days_with_arrears INT) AS $$
DECLARE
    v_first_day DATE := MAKE_DATE(p_year, p_month, 1);
    v_total_late_minutes INT := 0;
    v_days_with_arrears INT := 0;
    v_r RECORD;
    v_prev_checkin TIMESTAMP;
    v_company_id UUID;
    v_default_start TIME;
BEGIN
    -- Get user's company schedule settings
    SELECT company_id INTO v_company_id FROM users WHERE user_id = p_user_id;
    IF v_company_id IS NOT NULL THEN
        SELECT default_start_time INTO v_default_start
        FROM companies WHERE company_id = v_company_id;
    END IF;

    -- If no default_start configured, cannot calculate lateness
    IF v_default_start IS NULL THEN
        RETURN QUERY SELECT 0, 0;
        RETURN;
    END IF;

    -- Iterate through attendance logs pairing checkin->checkout
    FOR v_r IN
        SELECT timestamp, type FROM attendance_logs
        WHERE user_id = p_user_id AND timestamp >= v_first_day AND timestamp < v_first_day + INTERVAL '1 month'
        ORDER BY timestamp
    LOOP
        IF v_r.type = 'checkin' THEN
            v_prev_checkin := v_r.timestamp;
        ELSIF v_r.type = 'checkout' AND v_prev_checkin IS NOT NULL THEN
            -- Calculate minutes late: only if checkin was after default_start
            IF v_prev_checkin::TIME > v_default_start THEN
                v_total_late_minutes := v_total_late_minutes + EXTRACT(EPOCH FROM (v_prev_checkin::TIME - v_default_start)) / 60.0;
                v_days_with_arrears := v_days_with_arrears + 1;
            END IF;
            v_prev_checkin := NULL;
        END IF;
    END LOOP;

    RETURN QUERY SELECT v_total_late_minutes, v_days_with_arrears;
END;
$$ LANGUAGE plpgsql;

-- Trigger function to auto-update weekly hours summary
CREATE OR REPLACE FUNCTION trigger_update_weekly_hours()
RETURNS TRIGGER AS $$
DECLARE
    v_week_start DATE;
    v_total_hours NUMERIC;
    v_expected_hours NUMERIC := 40.0;
BEGIN
    v_week_start := DATE_TRUNC('week', NEW.timestamp)::DATE;

    v_total_hours := calculate_weekly_hours(NEW.user_id, v_week_start);

    INSERT INTO weekly_hours_summary (summary_id, user_id, week_start, total_hours, expected_hours, calculated_at)
    VALUES (gen_random_uuid(), NEW.user_id, v_week_start, v_total_hours, v_expected_hours, NOW())
    ON CONFLICT (user_id, week_start) DO UPDATE SET
        total_hours = EXCLUDED.total_hours,
        calculated_at = NOW();

    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Trigger function to auto-update monthly arrears summary
CREATE OR REPLACE FUNCTION trigger_update_monthly_arrears()
RETURNS TRIGGER AS $$
DECLARE
    v_year INT := EXTRACT(YEAR FROM NEW.timestamp)::INT;
    v_month INT := EXTRACT(MONTH FROM NEW.timestamp)::INT;
    v_arrears RECORD;
    v_summary_id UUID;
BEGIN
    SELECT * INTO v_arrears FROM calculate_monthly_arrears(NEW.user_id, v_year, v_month);

    SELECT summary_id INTO v_summary_id
    FROM monthly_arrears_summary
    WHERE user_id = NEW.user_id AND year = v_year AND month = v_month;

    IF v_summary_id IS NULL THEN
        INSERT INTO monthly_arrears_summary (summary_id, user_id, year, month, total_arrears_minutes, days_with_arrears)
        VALUES (gen_random_uuid(), NEW.user_id, v_year, v_month, v_arrears.total_arrears_minutes, v_arrears.days_with_arrears);
    ELSE
        UPDATE monthly_arrears_summary
        SET total_arrears_minutes = v_arrears.total_arrears_minutes,
            days_with_arrears = v_arrears.days_with_arrears
        WHERE summary_id = v_summary_id;
    END IF;

    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Create triggers on attendance_logs
CREATE TRIGGER trg_attendance_after_insert
    AFTER INSERT ON attendance_logs
    FOR EACH ROW
    EXECUTE FUNCTION trigger_update_weekly_hours();

CREATE TRIGGER trg_attendance_after_insert_arrears
    AFTER INSERT ON attendance_logs
    FOR EACH ROW
    WHEN (NEW.type IN ('checkin', 'checkout'))
    EXECUTE FUNCTION trigger_update_monthly_arrears();

-- Create index on companies.name for case-insensitive search
CREATE INDEX idx_companies_name ON companies(name);

-- Seed data: super-admin user (password: 2605admin)
INSERT INTO admin_users (admin_id, email, password, name, created_at, updated_at)
VALUES (
    '550e8400-e29b-41d4-a716-446655440003',
    'admin@turno.cl',
    '$2a$10$8mJubwXdec5BAyZZ6ywo8u7W6qy6OT7d7J9XqqdxfWMx3aaAwiMTO',
    'Administrador',
    NOW(),
    NOW()
) ON CONFLICT (admin_id) DO NOTHING;