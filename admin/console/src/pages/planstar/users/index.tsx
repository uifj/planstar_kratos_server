import { type User, services } from '@/services';
import {
  PageContainer,
  type ActionType,
  type ProColumns,
  ProTable,
} from '@ant-design/pro-components';
import { FormattedMessage, useAccess, useIntl } from '@umijs/max';
import { App, Popconfirm, Typography } from 'antd';
import React, { useRef } from 'react';
import {
  PAGE_SIZE,
  maskMobile,
  optionalIntFilter,
  useCursorPager,
} from '../shared';

type UserQuery = {
  keyword?: string;
  status?: string;
};

const UserTable: React.FC = () => {
  const actionRef = useRef<ActionType>(null);
  const pager = useCursorPager();
  const { canOperate } = useAccess();
  const { message } = App.useApp();
  const intl = useIntl();

  const changeStatus = async (record: User) => {
    const status = record.status === 1 ? 0 : 1;
    try {
      await services.console.UpdateUserStatus({ id: record.id, status });
      message.success(
        intl.formatMessage({ id: 'pages.planstar.users.updateSuccess' }),
      );
      actionRef.current?.reload();
    } catch {
      // 失败提示由 requestErrorConfig 统一处理。
    }
  };

  const columns: ProColumns<User>[] = [
    {
      title: <FormattedMessage id="pages.planstar.search.keyword" />,
      dataIndex: 'keyword',
      hideInTable: true,
      fieldProps: {
        placeholder: intl.formatMessage({
          id: 'pages.planstar.users.keywordPlaceholder',
        }),
      },
    },
    {
      title: 'ID',
      dataIndex: 'id',
      width: 90,
      search: false,
      copyable: true,
    },
    {
      title: <FormattedMessage id="pages.planstar.users.name" />,
      dataIndex: 'name',
      search: false,
      render: (_, record) =>
        record.name ? (
          <Typography.Text strong>{record.name}</Typography.Text>
        ) : (
          '-'
        ),
    },
    {
      title: <FormattedMessage id="pages.planstar.users.mobile" />,
      dataIndex: 'mobile',
      search: false,
      render: (_, record) => (
        <Typography.Text
          copyable={record.mobile ? { text: record.mobile } : false}
        >
          {maskMobile(record.mobile)}
        </Typography.Text>
      ),
    },
    {
      title: <FormattedMessage id="pages.planstar.users.status" />,
      dataIndex: 'status',
      width: 110,
      valueEnum: {
        1: {
          text: (
            <FormattedMessage id="pages.planstar.users.status.normal" />
          ),
          status: 'Success',
        },
        0: {
          text: (
            <FormattedMessage id="pages.planstar.users.status.disabled" />
          ),
          status: 'Default',
        },
      },
    },
    {
      title: <FormattedMessage id="pages.planstar.users.registeredAt" />,
      dataIndex: 'createdAt',
      valueType: 'dateTime',
      width: 200,
      search: false,
    },
    ...(canOperate
      ? ([
          {
            title: <FormattedMessage id="pages.planstar.operation" />,
            dataIndex: 'option',
            valueType: 'option',
            width: 100,
            search: false,
            render: (_, record) => {
              const disabling = record.status === 1;
              return [
                <Popconfirm
                  key="toggle"
                  title={intl.formatMessage({
                    id: disabling
                      ? 'pages.planstar.users.disableConfirm.title'
                      : 'pages.planstar.users.enableConfirm.title',
                  })}
                  description={intl.formatMessage({
                    id: disabling
                      ? 'pages.planstar.users.disableConfirm.description'
                      : 'pages.planstar.users.enableConfirm.description',
                  })}
                  onConfirm={() => changeStatus(record)}
                >
                  <a>
                    {intl.formatMessage({
                      id: disabling
                        ? 'pages.planstar.users.disable'
                        : 'pages.planstar.users.enable',
                    })}
                  </a>
                </Popconfirm>,
              ];
            },
          },
        ] as ProColumns<User>[])
      : []),
  ];

  return (
    <PageContainer
      title={intl.formatMessage({ id: 'pages.planstar.users.title' })}
      content={
        <Typography.Text type="secondary">
          <FormattedMessage id="pages.planstar.users.description" />
        </Typography.Text>
      }
    >
      <ProTable<User, UserQuery>
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
          const res = await services.console.ListUsers({
            keyword,
            status: optionalIntFilter(status),
            pageSize: PAGE_SIZE,
            pageToken,
          });
          const data = res.users ?? [];
          return {
            data,
            success: true,
            total: pager.total(current, data.length, res.nextPageToken),
          };
        }}
      />
    </PageContainer>
  );
};

export default UserTable;
