import { type OrderDetail, services } from '@/services';
import { FormattedMessage, useIntl, useRequest } from '@umijs/max';
import { Descriptions, Drawer, Table, Typography } from 'antd';
import React from 'react';
import {
  ORDER_STATUS,
  PAY_ORDER_STATUS,
  REFUND_STATUS,
  formatCents,
  statusText,
} from '../shared';

const OrderDrawer: React.FC<{ id?: number; onClose: () => void }> = ({
  id,
  onClose,
}) => {
  const intl = useIntl();
  const { data, loading } = useRequest(
    () => services.console.GetOrder({ id: id as number }),
    { ready: !!id, refreshDeps: [id], formatResult: (res: OrderDetail) => res },
  );

  return (
    <Drawer
      open={!!id}
      onClose={onClose}
      width={720}
      title={intl.formatMessage(
        { id: 'pages.planstar.orders.detail.title' },
        { id },
      )}
      loading={loading}
    >
      <Descriptions column={2} size="small" bordered>
        <Descriptions.Item label={<FormattedMessage id="pages.planstar.orders.orderNo" />} span={2}>
          {data?.order?.orderNo || '-'}
        </Descriptions.Item>
        <Descriptions.Item label={<FormattedMessage id="pages.planstar.orders.detail.subject" />} span={2}>
          {data?.title || '-'}
        </Descriptions.Item>
        <Descriptions.Item label={<FormattedMessage id="pages.planstar.userId" />}>
          {data?.order?.userId ?? '-'}
        </Descriptions.Item>
        <Descriptions.Item label={<FormattedMessage id="pages.planstar.orders.status" />}>
          {statusText(intl, ORDER_STATUS, data?.order?.status)}
        </Descriptions.Item>
        <Descriptions.Item label={<FormattedMessage id="pages.planstar.orders.amount" />}>
          {formatCents(data?.order?.amount)}
        </Descriptions.Item>
        <Descriptions.Item label={<FormattedMessage id="pages.planstar.orders.detail.totalAmount" />}>
          {formatCents(data?.totalAmount)}
        </Descriptions.Item>
        <Descriptions.Item label={<FormattedMessage id="pages.planstar.orders.detail.refundAmount" />}>
          {formatCents(data?.refundAmount)}
        </Descriptions.Item>
        <Descriptions.Item label={<FormattedMessage id="pages.planstar.orders.detail.payTime" />}>
          {data?.payTime
            ? new Date(data.payTime).toLocaleString()
            : '-'}
        </Descriptions.Item>
      </Descriptions>

      <Typography.Title level={5} style={{ marginTop: 16 }}>
        <FormattedMessage id="pages.planstar.orders.detail.payOrders" />
      </Typography.Title>
      <Table
        rowKey="id"
        size="small"
        dataSource={data?.payOrders ?? []}
        pagination={false}
        columns={[
          {
            title: intl.formatMessage({ id: 'pages.planstar.payOrders.payNo' }),
            dataIndex: 'payNo',
          },
          {
            title: intl.formatMessage({ id: 'pages.planstar.payOrders.channel' }),
            dataIndex: 'channel',
          },
          {
            title: intl.formatMessage({ id: 'pages.planstar.payOrders.amount' }),
            dataIndex: 'amount',
            align: 'right' as const,
            render: (v: number) => formatCents(v),
          },
          {
            title: intl.formatMessage({ id: 'pages.planstar.payOrders.status' }),
            dataIndex: 'status',
            render: (v: string) => statusText(intl, PAY_ORDER_STATUS, v),
          },
        ]}
      />

      <Typography.Title level={5} style={{ marginTop: 16 }}>
        <FormattedMessage id="pages.planstar.orders.detail.refunds" />
      </Typography.Title>
      <Table
        rowKey="id"
        size="small"
        dataSource={data?.refunds ?? []}
        pagination={false}
        columns={[
          {
            title: intl.formatMessage({ id: 'pages.planstar.refunds.refundNo' }),
            dataIndex: 'refundNo',
          },
          {
            title: intl.formatMessage({ id: 'pages.planstar.refunds.reasonCode' }),
            dataIndex: 'reasonCode',
          },
          {
            title: intl.formatMessage({ id: 'pages.planstar.refunds.amount' }),
            dataIndex: 'refundableAmount',
            align: 'right' as const,
            render: (v: number) => formatCents(v),
          },
          {
            title: intl.formatMessage({ id: 'pages.planstar.refunds.status' }),
            dataIndex: 'status',
            render: (v: string) => statusText(intl, REFUND_STATUS, v),
          },
        ]}
      />
    </Drawer>
  );
};

export default OrderDrawer;