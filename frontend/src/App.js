import React, { useState, useEffect } from 'react';
import axios from 'axios';
import { Table } from 'antd';
import ms from 'ms';

const App = () => {
  const [data, setData] = useState([]);
  const [loading, setLoading] = useState(false);

  const processData = (rawData) => {
    const aggregated = {};
    rawData.forEach(record => {
      const ip = record.ip;
      if (!aggregated[ip] || new Date(record.last_success) > new Date(aggregated[ip].last_success)) {
        aggregated[ip] = record;
      }
    });
    return Object.values(aggregated);
  };

  const fetchData = async () => {
    setLoading(true);
    try {
      const API_URL = process.env.REACT_APP_API_URL || 'http://localhost/api';
      const response = await axios.get(`${API_URL}/ping-data`);
      const aggregatedData = processData(response.data);
      setData(aggregatedData);
    } catch (error) {
      console.error('Ошибка получения данных', error);
    }
    setLoading(false);
  };

  useEffect(() => {
    fetchData();
    const interval = setInterval(fetchData, ms(process.env.REACT_APP_PING_INTERVAL))
    return () => clearInterval(interval);
  }, []);

  const columns = [
    {
      title: 'IP адрес',
      dataIndex: 'ip',
      key: 'ip',
    },
    {
      title: 'Ping (мс)',
      dataIndex: 'ping_time',
      key: 'ping_time',
      render: (ping_time) => (ping_time === -1 ? 'N/A' : ping_time),
    },
    {
      title: 'Дата последней попытки',
      dataIndex: 'last_success',
      key: 'last_success',
      render: (date) => new Date(date).toLocaleString(),
    },
  ];

  return (
    <div style={{ padding: '20px' }}>
      <h1>Статус пинга контейнеров</h1>
      <Table
        dataSource={data}
        columns={columns}
        rowKey="ip"  // Используем IP в качестве уникального идентификатора, чтобы не было дублей
        loading={loading}
        pagination={false}
      />
    </div>
  );
};

export default App;