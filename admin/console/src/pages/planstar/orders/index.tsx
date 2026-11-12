import { type Order, services } from '@/services';
import {
  PageContainer,
  type ActionType,
  type ProColumns,
  ProTable,
} from '@ant-design/pro-components';
import { FormattedMessage, useIntl } from '@umijs/max';
import { Typography } from 'antd';
import React, { useRef, useState } from 'react';
import {
  ORDER_STATUS,
  PAGE_SIZE,
  formatCents,
  optionalIntFilter,
  statusValueEnum,
  useCursorPager,
} from '../shared';
import OrderDrawer from './OrderDrawer';

type OrderQuery = {
  keyword?: string;
  status?: string;
};

const OrderTable: React.FC = () => {
  const actionRef = useRef<ActionType>(null);
  const pager = useCursorPager();
  const intl = useIntl();
  const [activeId, setActiveId] = useState<number>();

  const columns: ProColumns<Order>[] = [
    {
      title: <FormattedMessage id="pages.planstar.search.keyword" />,
      dataIndex: 'keyword',
      hideInTable: true,
      fieldProps: {
        placeholder: intl.formatMessage({
          id: 'pages.planstar.orders.keywordPlaceholder',
        }),
      },
    },
    {
      title: <FormattedMessage id="pages.planstar.orders.orderNo" />,
      dataIndex: 'orderNo',
      copyable: true,
      width: 220,
      ellipsis: true,
      search: false,
      render: (_, record) => record.orderNo || '-',
    },
    {
      title: <FormattedMessage id="pages.planstar.orders.userId" />,
      dataIndex: 'userId',
      width: 110,
      search: false,
      render: (_, record) => record.userId ?? '-',
    },
    {
      title: <FormattedMessage id="pages.planstar.orders.amount" />,
      dataIndex: 'amount',
      width: 130,
      align: 'right',
      search: false,
      render: (_, record) => formatCents(record.amount),
    },
    {
      title: <FormattedMessage id="pages.planstar.orders.status" />,
      dataIndex: 'status',
      width: 120,
      valueEnum: statusValueEnum(intl, ORDER_STATUS),
    },
    {
      title: <FormattedMessage id="pages.planstar.orders.createdAt" />,
      dataIndex: 'createdAt',
      valueType: 'dateTime',
      width: 200,
      search: false,
    },
    {
      title: <FormattedMessage id="pages.planstar.operation" />,
      dataIndex: 'option',
      valueType: 'option',
      width: 80,
      search: false,
      render: (_, record) => [
        <a key="detail" onClick={() => setActiveId(record.id)}>
          <FormattedMessage id="pages.planstar.detail" />
        </a>,
      ],
    },
  ];

  return (
    <PageContainer
      title={intl.formatMessage({ id: 'pages.planstar.orders.title' })}
      content={
        <Typography.Text type="secondary">
          <FormattedMessage id="pages.planstar.orders.description" />
        </Typography.Text>
      }
    >
      <ProTable<Order, OrderQuery>
        actionRef={actionRef}
        rowKey="id"
        cardBordered
        columns={columns}
        search={{ labelWidth: 'auto' }}
        options={{ density: false, fullScreen: false, setting: false }}
        pagination={{ pageSize: PAGE_SIZE, showSizeChanger: false }}
        request={async (params) => {
          const keyword = params.keyword?.trim() || undefined;
          const status =
            params.status === undefined || params.status === ''
              ? undefined
              : Number(params.status);
          const current = params.current ?? 1;
          const pageToken = pager.pageToken(
            current,
            `${keyword ?? ''}|${status ?? ''}`,
          );
          const res = await services.console.ListOrders({
            keyword,
            status: optionalIntFilter(status),
            pageSize: PAGE_SIZE,
            pageToken,
          });
          const data = res.orders ?? [];
          return {
            data,
            success: true,
            total: pager.total(current, data.length, res.nextPageToken),
          };
        }}
      />
      <OrderDrawer id={activeId} onClose={() => setActiveId(undefined)} />
    </PageContainer>
  );
};

export default OrderTable;
