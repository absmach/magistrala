// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package atom

import "context"

type defaultDeviceType struct {
	DeviceType DeviceType
	Version    DeviceTypeVersion
}

// defaultDeviceTypes is the small catalogue every tenant starts with. The
// versions deliberately avoid required measurements: Atom validates device
// attributes, not telemetry messages, so a default type must not block creating
// a device before attributes are populated.
func defaultDeviceTypes(tenantID string) []defaultDeviceType {
	return []defaultDeviceType{
		defaultType(tenantID, "water-meter", "Water Meter", "Measures water consumption and flow.", measurements(
			numberMeasurement("total_volume", "m3"),
			numberMeasurement("flow_rate", "m3/h"),
			numberMeasurement("battery", "%", withRange(0, 100)),
		), intervalCommand()),
		defaultType(tenantID, "pressure-sensor", "Pressure Sensor", "Measures line, wellhead, and casing pressure.", measurements(
			numberMeasurement("pressure", "bar"),
			numberMeasurement("wellhead_pressure", "psi"),
			numberMeasurement("casing_pressure", "psi"),
			numberMeasurement("battery", "%", withRange(0, 100)),
		), intervalCommand()),
		defaultType(tenantID, "energy-meter", "Energy Meter", "Measures electrical energy and power draw.", measurements(
			numberMeasurement("energy", "kWh"),
			numberMeasurement("power", "W"),
			numberMeasurement("voltage", "V"),
		), intervalCommand()),
		defaultType(tenantID, "pump-controller", "Pump Controller", "Controls and monitors pump operation.", measurements(
			booleanMeasurement("pump_running"),
			numberMeasurement("pump_energy", "kWh"),
			numberMeasurement("battery", "%", withRange(0, 100)),
		), command("set_pump", "Set pump state.", map[string]string{"running": ValueTypeBoolean})),
	}
}

func (c *Client) seedDefaultDeviceTypes(ctx context.Context, tenantID string) error {
	existing, err := c.ListDeviceTypes(ctx, DeviceTypeQuery{TenantID: tenantID, Limit: 1000})
	if err != nil {
		return err
	}
	byKey := make(map[string]DeviceType, len(existing.Items))
	for _, deviceType := range existing.Items {
		byKey[deviceType.Key] = deviceType
	}

	for _, seed := range defaultDeviceTypes(tenantID) {
		deviceType, ok := byKey[seed.DeviceType.Key]
		if !ok {
			created, err := c.CreateDeviceType(ctx, seed.DeviceType)
			if err != nil {
				return err
			}
			deviceType = created
		}

		versions, err := c.listDeviceTypeVersionsRaw(ctx, deviceType.ID)
		if err != nil {
			return err
		}
		if len(versions) > 0 {
			continue
		}
		if _, err := c.CreateDeviceTypeVersion(ctx, deviceType.ID, seed.Version); err != nil {
			return err
		}
	}
	return nil
}

func defaultType(tenantID, key, name, description string, measurements []Measurement, commands ...Command) defaultDeviceType {
	return defaultDeviceType{
		DeviceType: DeviceType{
			TenantID:    tenantID,
			Key:         key,
			Name:        name,
			Description: description,
			Status:      DeviceTypeStatusActive,
		},
		Version: DeviceTypeVersion{
			Version: 1,
			Status:  DeviceTypeVersionStatusActive,
			Capabilities: CapabilityDocument{
				Measurements: measurements,
				Commands:     compactCommands(commands),
			},
		},
	}
}

func measurements(measurements ...Measurement) []Measurement {
	return measurements
}

type measurementOption func(*Measurement)

func numberMeasurement(name, unit string, opts ...measurementOption) Measurement {
	measurement := Measurement{Name: name, Unit: unit, Type: ValueTypeNumber, Access: MeasurementAccessRead}
	for _, opt := range opts {
		opt(&measurement)
	}
	return measurement
}

func booleanMeasurement(name string) Measurement {
	return Measurement{Name: name, Type: ValueTypeBoolean, Access: MeasurementAccessRead}
}

func withRange(min, max float64) measurementOption {
	return func(measurement *Measurement) {
		measurement.Min = &min
		measurement.Max = &max
	}
}

func command(name, description string, params map[string]string) Command {
	return Command{Name: name, Description: description, Params: params}
}

func intervalCommand() Command {
	return command("set_interval", "Set telemetry reporting interval.", map[string]string{"seconds": ValueTypeInteger})
}

func compactCommands(commands []Command) []Command {
	compacted := commands[:0]
	for _, command := range commands {
		if command.Name != "" {
			compacted = append(compacted, command)
		}
	}
	return compacted
}
