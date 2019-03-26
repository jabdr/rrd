extern char *rrdCreate(const char *filename, unsigned long step, time_t start, int argc, const char **argv);
extern char *rrdUpdate(const char *filename, const char *template, int argc, const char **argv);
extern char *rrdGraph(rrd_info_t **ret, int argc, char **argv);
extern char *rrdInfo(rrd_info_t **ret, char *filename);
extern char *rrdFetch(int *ret, char *filename, const char *cf, time_t *start, time_t *end, unsigned long *step, unsigned long *ds_cnt, char ***ds_namv, double **data);
extern char *rrdXport(int *ret, int argc, char **argv, int *xsize, time_t *start, time_t *end, unsigned long *step, unsigned long *col_cnt, char ***legend_v, double **data);
extern char *arrayGetCString(char **values, int i);


typedef struct rrd_dumper_t rrd_dumper_t;

typedef struct {
    time_t timestamp;
    int num_cols;
    double *cols;
} rrd_row_dump_t;

extern rrd_dumper_t* init_rrd_dumper(char *filename);
extern int seek_rrd_dumper_to_def(rrd_dumper_t* dumper, char *rra_definition_name);
extern rrd_row_dump_t* yield_rrd_dumper_row(rrd_dumper_t* dumper);
extern void close_rrd_dumper(rrd_dumper_t* dumper);

