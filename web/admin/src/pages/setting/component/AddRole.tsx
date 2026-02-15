import {
  Box,
  Tooltip,
  Stack,
  Select,
  MenuItem,
  Radio,
  Button,
  TextField,
} from '@mui/material';
import { getApiV1UserList, postApiV1UserCreate } from '@/request/User';
import { postApiV1KnowledgeBaseUserInvite } from '@/request/KnowledgeBase';
import {
  ConstsUserKBPermission,
  ConstsUserRole,
  V1KBUserInviteReq,
  V1UserListItemResp,
} from '@/request/types';
import { FormItem } from '@/components/Form';
import NoData from '@/assets/images/nodata.png';
import Card from '@/components/Card';
import { message, Modal, Table } from '@ctzhian/ui';
import dayjs from 'dayjs';
import { ColumnType } from '@ctzhian/ui/dist/Table';
import { useEffect, useMemo, useState } from 'react';
import { useAppSelector } from '@/store';
import { VersionCanUse } from '@/components/VersionMask';
import { PROFESSION_VERSION_PERMISSION } from '@/constant/version';
import { copyText, generatePassword } from '@/utils';

interface AddRoleProps {
  open: boolean;
  onCancel: () => void;
  onOk: () => void;
  selectedIds: string[];
}

const AddRole = ({ open, onCancel, onOk, selectedIds }: AddRoleProps) => {
  const { kb_id } = useAppSelector(state => state.config);
  const { license } = useAppSelector(state => state.config);
  const [list, setList] = useState<V1UserListItemResp[]>([]);
  const [loading, setLoading] = useState(false);
  const [selectedRowKeys, setSelectedRowKeys] = useState<string>('');
  const [perm, setPerm] = useState<V1KBUserInviteReq['perm']>(
    ConstsUserKBPermission.UserKBPermissionFullControl,
  );
  const [createOpen, setCreateOpen] = useState(false);
  const [createLoading, setCreateLoading] = useState(false);
  const [newAccount, setNewAccount] = useState('');
  const [createdAccount, setCreatedAccount] = useState('');
  const [createdPassword, setCreatedPassword] = useState('');

  const columns: ColumnType<V1UserListItemResp>[] = [
    {
      title: '',
      dataIndex: 'id',
      width: 80,
      render: (text: string) => (
        <Tooltip
          arrow
          placement='top'
          title={selectedIds.includes(text) ? '已添加' : ''}
        >
          <span>
            <Radio
              disableRipple
              size='small'
              disabled={selectedIds.includes(text)}
              checked={selectedRowKeys === text}
              onChange={() => {
                setSelectedRowKeys(text);
              }}
              sx={{
                '.MuiTouchRipple-root': {
                  display: 'none',
                },
              }}
            />
          </span>
        </Tooltip>
      ),
    },
    {
      title: '用户名',
      dataIndex: 'account',
      render: (text: string) => (
        <Stack direction={'row'} alignItems={'center'} gap={2}>
          {text}
        </Stack>
      ),
    },
    {
      title: '上次使用时间',
      dataIndex: 'last_access',
      render: (text: string) => (
        <Box>{text ? dayjs(text).format('YYYY-MM-DD HH:mm:ss') : '-'}</Box>
      ),
    },
  ];
  const getData = () => {
    setLoading(true);
    getApiV1UserList()
      .then(res => {
        setList(res.users || []);
      })
      .finally(() => {
        setLoading(false);
      });
  };

  const onSubmit = () => {
    if (!selectedRowKeys) {
      message.error('请选择用户');
      return;
    }
    postApiV1KnowledgeBaseUserInvite({
      kb_id,
      user_id: selectedRowKeys,
      perm,
    }).then(() => {
      onOk();
      message.success('添加成功');
    });
  };

  const onCreateUser = () => {
    const account = newAccount.trim();
    if (!account) {
      message.error('请输入用户名');
      return;
    }

    const password = generatePassword(12);
    setCreateLoading(true);
    postApiV1UserCreate({
      account,
      password,
      role: ConstsUserRole.UserRoleUser,
    })
      .then(res => {
        message.success('用户创建成功');
        setCreateOpen(false);
        setNewAccount('');
        setCreatedAccount(account);
        setCreatedPassword(password);
        getData();
        const createdID = (res as any)?.id || (res as any)?.data?.id;
        if (createdID) {
          setSelectedRowKeys(createdID);
        }
      })
      .finally(() => {
        setCreateLoading(false);
      });
  };

  const onCopyUserInfo = () => {
    copyText(
      `用户名: ${createdAccount}\n密码: ${createdPassword}`,
      () => {
        setCreatedAccount('');
        setCreatedPassword('');
      },
      1.5,
      '，请妥善保存',
    );
  };

  useEffect(() => {
    if (open) {
      getData();
    } else {
      setSelectedRowKeys('');
      setPerm(
        ConstsUserKBPermission.UserKBPermissionFullControl as V1KBUserInviteReq['perm'],
      );
    }
  }, [open]);

  const isPro = useMemo(() => {
    return PROFESSION_VERSION_PERMISSION.includes(license.edition!);
  }, [license.edition]);

  return (
    <Modal
      title='添加 Wiki 站管理员'
      open={open}
      onCancel={onCancel}
      onOk={onSubmit}
      width={800}
    >
      <Card
        sx={{
          py: 2,
          border: '1px solid',
          borderColor: 'divider',
        }}
      >
        <Table
          columns={columns}
          dataSource={list}
          rowKey='id'
          size='small'
          updateScrollTop={false}
          sx={{
            '.MuiTableContainer-root': {
              maxHeight: 'calc(100vh - 370px)',
              minHeight: 200,
            },
            '& .MuiTableCell-root': {
              height: 40,
              '&:first-of-type': {
                pl: 2,
              },
            },
            '.MuiTableHead-root .cx-selection-column .MuiCheckbox-root': {
              visibility: 'hidden',
            },
          }}
          pagination={false}
          // rowSelection={{
          //   hideSelectAll: true,
          //   selectedRowKeys: selectedRowKeys,
          //   getCheckboxProps: (record: V1UserListItemResp) => {
          //     return {
          //       disabled:
          //         selectedRowKeys.length > 0
          //           ? !selectedRowKeys.includes(record.id!)
          //           : false,
          //     };
          //   },
          //   // @ts-expect-error 类型错误
          //   onChange: (selectedRowKeys: string[]) => {
          //     setSelectedRowKeys(selectedRowKeys);
          //   },
          // }}
          renderEmpty={
            loading ? (
              <Box></Box>
            ) : (
              <Stack alignItems={'center'}>
                <img src={NoData} width={150} />
                <Box
                  sx={{
                    fontSize: 12,
                    lineHeight: '20px',
                    color: 'text.tertiary',
                  }}
                >
                  暂无数据
                </Box>
              </Stack>
            )
          }
        />
      </Card>
      <Stack
        direction='row'
        justifyContent='flex-end'
        sx={{
          mt: 1,
        }}
      >
        <Button
          size='small'
          variant='outlined'
          onClick={() => {
            setCreateOpen(true);
          }}
        >
          新建用户
        </Button>
      </Stack>
      <FormItem
        label={
          <Stack
            sx={{ display: 'inline-flex' }}
            direction={'row'}
            alignItems={'center'}
            gap={0.5}
          >
            权限
          </Stack>
        }
        sx={{ mt: 2 }}
      >
        <Select
          fullWidth
          sx={{ height: 52 }}
          value={perm}
          MenuProps={{
            sx: {
              '.Mui-disabled': {
                opacity: '1 !important',
                color: 'text.disabled',
              },
            },
          }}
          onChange={e => setPerm(e.target.value as V1KBUserInviteReq['perm'])}
        >
          <MenuItem value={ConstsUserKBPermission.UserKBPermissionFullControl}>
            完全控制
          </MenuItem>

          <MenuItem
            value={ConstsUserKBPermission.UserKBPermissionDocManage}
            disabled={!isPro}
          >
            文档管理{' '}
            <VersionCanUse permission={PROFESSION_VERSION_PERMISSION} />
          </MenuItem>
          <MenuItem
            value={ConstsUserKBPermission.UserKBPermissionDataOperate}
            disabled={!isPro}
          >
            数据运营{' '}
            <VersionCanUse permission={PROFESSION_VERSION_PERMISSION} />
          </MenuItem>
        </Select>
      </FormItem>

      <Modal
        title='创建新用户'
        open={createOpen}
        onCancel={() => {
          setCreateOpen(false);
          setNewAccount('');
        }}
        onOk={onCreateUser}
        okButtonProps={{ loading: createLoading }}
      >
        <FormItem label='用户名' required>
          <TextField
            fullWidth
            autoFocus
            value={newAccount}
            placeholder='请输入用户名'
            onChange={e => setNewAccount(e.target.value)}
          />
        </FormItem>
        <Box sx={{ fontSize: 12, color: 'text.tertiary', mt: 1 }}>
          系统会自动生成初始密码，创建成功后请复制保存。
        </Box>
      </Modal>

      <Modal
        title='用户创建成功'
        open={!!createdPassword}
        closable={false}
        cancelText='关闭'
        onCancel={() => {
          setCreatedAccount('');
          setCreatedPassword('');
        }}
        okText='复制账号密码'
        onOk={onCopyUserInfo}
      >
        <Card sx={{ p: 2, fontSize: 14, bgcolor: 'background.paper3' }}>
          <Stack direction='row'>
            <Box sx={{ width: 80 }}>用户名</Box>
            <Box sx={{ fontWeight: 700 }}>{createdAccount}</Box>
          </Stack>
          <Stack direction='row' sx={{ mt: 1 }}>
            <Box sx={{ width: 80 }}>密码</Box>
            <Box sx={{ fontWeight: 700 }}>{createdPassword}</Box>
          </Stack>
        </Card>
      </Modal>
    </Modal>
  );
};

export default AddRole;
