# EME_Base
Earth Moon Earth Base Station

*Temperature readout:*
- The remote sensor just reads the A/D voltage and sends it to the base station
- Max range on the remote A/D is 8 bits or 1023
- Reference voltage into the A/D is 4.99 volts precision reference LM335
- At the base station, the raw number is convertaed to a voltage

$$V_o = \frac{4.99}{1023} * remote-sensor-read}$$ (1)

- Temperature sensor is referenced to zero degrees Kelvin (-273 deg C)
- Sensor output voltage at 298 deg K (25 deg C) is 2.98 volts
- Sensor sensitivity is 10 mv/degK or 0.01 v/degK

$$V_o = aT + b$$
$$2.98 = 0.01 * 298 + b$$

- Hence b = 0 and:

$$T = \frac{V_o}{0.01} = 100 * V_o$$ (2)

- Combining equations (1) and (2):

$$T = \frac{499}{1023} * remote_sensor_read$$

- A/D reference voltage and max A/D range are saved in adjust.yaml
- Absolute zero, clibration votage and calibration temperature are also saved in adjust.yaml
- Airfactor and sinkfactor are just fudge factors that are mutiplied by the read sensor value

*Power Readout*

- The forward voltage signal from the amp is non linear
- Until we collect the data, I will use the shape from the 1KW amp
- But adjust it to 5.02 volts at 1341 W from the 1.2 KW amp
- Curve fitting to a second order polynomial, we get the following
- I am not using zero intercept because it results in very odd data

$$O_o = -3.0e^{-6}V_o^{2}+0.0081V_o+1.3066$$

- Where $V_o$ is obtained from quaition (1)


