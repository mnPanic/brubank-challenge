# brubank-challenge

Challenge de entrevista de Brubank, Febrero 2023

TODOs:

- Validar formato números de teléfono

Notas sobre el servicio de consulta de usuarios

https://interview-brubank-api.herokuapp.com/users/:phoneNumber

Dudas:

- Qué pasa si se llama a si mismo?
- Puede tener amigos internacionales o solo nacionales?
- Qué hacer con las llamadas que tengan un número origen que no sea el del
  usuario para generar factura. Error? Ignorar?
- Qué debería pasar con las llamadas que están fuera del período de facturación?
- Fecha de llamadas del CSV en UTC, qué pasa si no está en ese timezone? Se
  convierte? Se devuelve un error?

Notas de decisiones y separación de responsabilidades

- El hecho de que las llamadas vengan en un CSV es accidental, bien podría ser
  un array en un JSON. Por esa razón la lógica de parseo queda del lado del
  handler.
- Para los assertions usé testify, que es una gran lib.